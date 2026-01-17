package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/huylvt/gisty/internal/model"
	"github.com/huylvt/gisty/internal/repository"
)

var (
	// ErrInvalidExpiresIn is returned when the expires_in value is invalid
	ErrInvalidExpiresIn = errors.New("paste: invalid expires_in value")
	// ErrContentTooLarge is returned when content exceeds max size
	ErrContentTooLarge = errors.New("paste: content too large")
	// ErrEmptyContent is returned when content is empty
	ErrEmptyContent = errors.New("paste: content cannot be empty")
	// ErrInvalidSyntaxType is returned when syntax type is not in whitelist
	ErrInvalidSyntaxType = errors.New("paste: invalid syntax type")
	// ErrPasteNotFound is returned when paste is not found
	ErrPasteNotFound = errors.New("paste: not found")
	// ErrPasteExpired is returned when paste has expired
	ErrPasteExpired = errors.New("paste: expired")
)

const (
	// MaxContentSize is the maximum allowed content size (1MB)
	MaxContentSize = 1 * 1024 * 1024
	// DefaultSyntaxType is the default syntax type for pastes
	DefaultSyntaxType = "plaintext"
)

// ValidSyntaxTypes is a whitelist of allowed syntax types
var ValidSyntaxTypes = map[string]bool{
	"":           true, // empty is allowed (will use default)
	"text":       true,
	"plaintext":  true,
	"markdown":   true,
	"json":       true,
	"xml":        true,
	"html":       true,
	"css":        true,
	"javascript": true,
	"typescript": true,
	"python":     true,
	"go":         true,
	"golang":     true,
	"java":       true,
	"c":          true,
	"cpp":        true,
	"csharp":     true,
	"ruby":       true,
	"php":        true,
	"rust":       true,
	"swift":      true,
	"kotlin":     true,
	"scala":      true,
	"sql":        true,
	"bash":       true,
	"shell":      true,
	"powershell": true,
	"yaml":       true,
	"toml":       true,
	"ini":        true,
	"dockerfile": true,
	"makefile":   true,
	"nginx":      true,
	"apache":     true,
	"lua":        true,
	"perl":       true,
	"r":          true,
	"matlab":     true,
	"latex":      true,
	"diff":       true,
	"graphql":    true,
	"protobuf":   true,
	"haskell":    true,
	"elixir":     true,
	"erlang":     true,
	"clojure":    true,
	"lisp":       true,
	"vim":        true,
	"assembly":   true,
}

// CreatePasteRequest represents the request to create a new paste
type CreatePasteRequest struct {
	Content    string `json:"content" binding:"required"`
	SyntaxType string `json:"syntax_type"`
	ExpiresIn  string `json:"expires_in"` // "10m", "1h", "1d", "1w", "never", "burn"
	IsPrivate  bool   `json:"is_private"`
}

// CreatePasteResponse represents the response after creating a paste
type CreatePasteResponse struct {
	ShortID   string  `json:"short_id"`
	URL       string  `json:"url"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

// GetPasteResponse represents the response when retrieving a paste
type GetPasteResponse struct {
	ShortID    string  `json:"short_id"`
	Content    string  `json:"content"`
	SyntaxType string  `json:"syntax_type"`
	CreatedAt  string  `json:"created_at"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
}

// PasteService handles paste business logic
type PasteService struct {
	kgs            *KGS
	storage        *Storage
	cache          *Cache
	pasteRepo      *repository.PasteRepository
	syntaxDetector *SyntaxDetector
	baseURL        string
}

// NewPasteService creates a new PasteService
func NewPasteService(kgs *KGS, storage *Storage, cache *Cache, pasteRepo *repository.PasteRepository, baseURL string) *PasteService {
	return &PasteService{
		kgs:            kgs,
		storage:        storage,
		cache:          cache,
		pasteRepo:      pasteRepo,
		syntaxDetector: NewSyntaxDetector(),
		baseURL:        baseURL,
	}
}

// CreatePaste creates a new paste
func (s *PasteService) CreatePaste(ctx context.Context, req *CreatePasteRequest) (*CreatePasteResponse, error) {
	// Validate content
	if len(req.Content) == 0 {
		return nil, ErrEmptyContent
	}
	if len(req.Content) > MaxContentSize {
		return nil, ErrContentTooLarge
	}

	// Normalize and validate syntax type
	syntaxType := strings.ToLower(strings.TrimSpace(req.SyntaxType))
	if !ValidSyntaxTypes[syntaxType] {
		return nil, ErrInvalidSyntaxType
	}
	if syntaxType == "" {
		// Auto-detect language from content
		syntaxType = s.syntaxDetector.DetectLanguage(req.Content)
	}

	// Parse expiration
	expiresAt, burnAfterRead, err := s.parseExpiration(req.ExpiresIn)
	if err != nil {
		return nil, err
	}

	// Get a unique short ID from KGS
	shortID, err := s.kgs.GetNextKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("paste: failed to get short ID: %w", err)
	}

	// Save content to S3
	if err := s.storage.SaveContent(ctx, shortID, req.Content); err != nil {
		return nil, fmt.Errorf("paste: failed to save content: %w", err)
	}

	// Create paste record in MongoDB
	paste := &model.Paste{
		ShortID:       shortID,
		ContentKey:    s.storage.buildKey(shortID),
		ExpiresAt:     expiresAt,
		CreatedAt:     time.Now(),
		SyntaxType:    syntaxType,
		IsPrivate:     req.IsPrivate,
		BurnAfterRead: burnAfterRead,
	}

	if err := s.pasteRepo.Create(ctx, paste); err != nil {
		// Try to clean up S3 on failure
		s.storage.DeleteContent(ctx, shortID)
		return nil, fmt.Errorf("paste: failed to create record: %w", err)
	}

	// Cache the content (optional, best effort)
	cacheTTL := DefaultCacheTTL
	if expiresAt != nil {
		ttl := time.Until(*expiresAt)
		if ttl > 0 && ttl < cacheTTL {
			cacheTTL = ttl
		}
	}
	// Don't cache burn-after-read pastes
	if !burnAfterRead {
		_ = s.cache.Set(ctx, shortID, req.Content, cacheTTL)
	}

	// Build response
	response := &CreatePasteResponse{
		ShortID: shortID,
		URL:     s.buildURL(shortID),
	}

	if expiresAt != nil {
		formatted := expiresAt.Format(time.RFC3339)
		response.ExpiresAt = &formatted
	}

	return response, nil
}

// parseExpiration parses the expires_in string and returns expiration time
func (s *PasteService) parseExpiration(expiresIn string) (*time.Time, bool, error) {
	if expiresIn == "" || expiresIn == "never" {
		return nil, false, nil
	}

	if expiresIn == "burn" {
		return nil, true, nil
	}

	// Parse duration-like strings
	var duration time.Duration
	switch expiresIn {
	case "10m":
		duration = 10 * time.Minute
	case "30m":
		duration = 30 * time.Minute
	case "1h":
		duration = 1 * time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "12h":
		duration = 12 * time.Hour
	case "1d":
		duration = 24 * time.Hour
	case "3d":
		duration = 3 * 24 * time.Hour
	case "1w":
		duration = 7 * 24 * time.Hour
	case "2w":
		duration = 14 * 24 * time.Hour
	case "1M":
		duration = 30 * 24 * time.Hour
	default:
		// Try to parse as Go duration
		var err error
		duration, err = time.ParseDuration(expiresIn)
		if err != nil {
			return nil, false, ErrInvalidExpiresIn
		}
	}

	expiresAt := time.Now().Add(duration)
	return &expiresAt, false, nil
}

// buildURL constructs the full URL for a paste
func (s *PasteService) buildURL(shortID string) string {
	return s.baseURL + "/" + shortID
}

// GetPaste retrieves a paste by its short ID
func (s *PasteService) GetPaste(ctx context.Context, shortID string) (*GetPasteResponse, error) {
	// Get paste metadata from MongoDB
	paste, err := s.pasteRepo.GetByShortID(ctx, shortID)
	if err != nil {
		if errors.Is(err, repository.ErrPasteNotFound) {
			return nil, ErrPasteNotFound
		}
		return nil, fmt.Errorf("paste: failed to get paste: %w", err)
	}

	// Check if paste has expired
	if paste.IsExpired() {
		// Clean up expired paste (best effort)
		go s.deletePaste(context.Background(), shortID)
		return nil, ErrPasteExpired
	}

	// Try to get content from cache first
	content, found, err := s.cache.Get(ctx, shortID)
	if err != nil {
		// Log error but continue to fetch from storage
		found = false
	}

	// Cache miss - fetch from S3
	if !found {
		content, err = s.storage.GetContent(ctx, shortID)
		if err != nil {
			if errors.Is(err, ErrContentNotFound) {
				return nil, ErrPasteNotFound
			}
			return nil, fmt.Errorf("paste: failed to get content: %w", err)
		}

		// Update cache (best effort, don't cache burn-after-read)
		if !paste.BurnAfterRead {
			cacheTTL := DefaultCacheTTL
			if paste.ExpiresAt != nil {
				ttl := time.Until(*paste.ExpiresAt)
				if ttl > 0 && ttl < cacheTTL {
					cacheTTL = ttl
				}
			}
			_ = s.cache.Set(ctx, shortID, content, cacheTTL)
		}
	}

	// Handle burn after read
	if paste.BurnAfterRead {
		// Delete the paste after reading (async to not block response)
		go s.deletePaste(context.Background(), shortID)
	}

	// Build response
	response := &GetPasteResponse{
		ShortID:    paste.ShortID,
		Content:    content,
		SyntaxType: paste.SyntaxType,
		CreatedAt:  paste.CreatedAt.Format(time.RFC3339),
	}

	if paste.ExpiresAt != nil {
		formatted := paste.ExpiresAt.Format(time.RFC3339)
		response.ExpiresAt = &formatted
	}

	return response, nil
}

// DeletePaste removes a paste by its short ID
func (s *PasteService) DeletePaste(ctx context.Context, shortID string) error {
	// Check if paste exists first
	_, err := s.pasteRepo.GetByShortID(ctx, shortID)
	if err != nil {
		if errors.Is(err, repository.ErrPasteNotFound) {
			return ErrPasteNotFound
		}
		return fmt.Errorf("paste: failed to get paste: %w", err)
	}

	// Delete from all layers
	s.deletePaste(ctx, shortID)

	return nil
}

// deletePaste removes a paste from all storage layers (internal helper)
func (s *PasteService) deletePaste(ctx context.Context, shortID string) {
	// Delete from cache
	_ = s.cache.Delete(ctx, shortID)
	// Delete from S3
	_ = s.storage.DeleteContent(ctx, shortID)
	// Delete from MongoDB
	_ = s.pasteRepo.Delete(ctx, shortID)
}