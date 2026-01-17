package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huylvt/gisty/internal/service"
)

// PasteHandler handles paste-related HTTP requests
type PasteHandler struct {
	pasteService *service.PasteService
}

// NewPasteHandler creates a new PasteHandler
func NewPasteHandler(pasteService *service.PasteService) *PasteHandler {
	return &PasteHandler{
		pasteService: pasteService,
	}
}

// CreatePasteRequest represents the request body for creating a paste
type CreatePasteRequest struct {
	Content    string `json:"content" binding:"required" example:"console.log('Hello, World!')"`
	SyntaxType string `json:"syntax_type" example:"javascript"`
	ExpiresIn  string `json:"expires_in" example:"1h"`
	IsPrivate  bool   `json:"is_private" example:"false"`
}

// CreatePasteResponse represents the response after creating a paste
type CreatePasteResponse struct {
	ShortID   string  `json:"short_id" example:"xK9a2B"`
	URL       string  `json:"url" example:"http://localhost:8080/xK9a2B"`
	ExpiresAt *string `json:"expires_at,omitempty" example:"2024-01-15T15:00:00Z"`
}

// GetPasteResponse represents the response when retrieving a paste
type GetPasteResponse struct {
	ShortID    string  `json:"short_id" example:"xK9a2B"`
	Content    string  `json:"content" example:"console.log('Hello, World!')"`
	SyntaxType string  `json:"syntax_type" example:"javascript"`
	CreatedAt  string  `json:"created_at" example:"2024-01-15T14:00:00Z"`
	ExpiresAt  *string `json:"expires_at,omitempty" example:"2024-01-15T15:00:00Z"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Paste not found"`
	MaxSize string `json:"max_size,omitempty" example:"1MB"`
}

// CreatePaste godoc
// @Summary Create a new paste
// @Description Create a new code/text snippet with optional expiration and syntax highlighting
// @Tags pastes
// @Accept json
// @Produce json
// @Param request body CreatePasteRequest true "Paste content and options"
// @Success 201 {object} CreatePasteResponse "Paste created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request (empty content, invalid syntax_type, invalid expires_in)"
// @Failure 413 {object} ErrorResponse "Content too large (max 1MB)"
// @Failure 429 {object} ErrorResponse "Rate limit exceeded"
// @Failure 503 {object} ErrorResponse "Service temporarily unavailable"
// @Router /pastes [post]
func (h *PasteHandler) CreatePaste(c *gin.Context) {
	var req service.CreatePasteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	response, err := h.pasteService.CreatePaste(c.Request.Context(), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetPaste godoc
// @Summary Get a paste by ID
// @Description Retrieve a paste's content and metadata by its short ID
// @Tags pastes
// @Accept json
// @Produce json
// @Param id path string true "Paste short ID" example(xK9a2B)
// @Success 200 {object} GetPasteResponse "Paste retrieved successfully"
// @Failure 400 {object} ErrorResponse "Missing paste ID"
// @Failure 404 {object} ErrorResponse "Paste not found"
// @Failure 410 {object} ErrorResponse "Paste has expired"
// @Router /pastes/{id} [get]
func (h *PasteHandler) GetPaste(c *gin.Context) {
	shortID := c.Param("id")
	if shortID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing paste ID",
		})
		return
	}

	response, err := h.pasteService.GetPaste(c.Request.Context(), shortID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeletePaste godoc
// @Summary Delete a paste
// @Description Delete a paste by its short ID
// @Tags pastes
// @Accept json
// @Produce json
// @Param id path string true "Paste short ID" example(xK9a2B)
// @Success 204 "Paste deleted successfully"
// @Failure 400 {object} ErrorResponse "Missing paste ID"
// @Failure 404 {object} ErrorResponse "Paste not found"
// @Router /pastes/{id} [delete]
func (h *PasteHandler) DeletePaste(c *gin.Context) {
	shortID := c.Param("id")
	if shortID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing paste ID",
		})
		return
	}

	err := h.pasteService.DeletePaste(c.Request.Context(), shortID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ShortURL handles GET /:id with content negotiation
// Returns JSON for Accept: application/json, plain text otherwise
func (h *PasteHandler) ShortURL(c *gin.Context) {
	shortID := c.Param("id")
	if shortID == "" {
		c.String(http.StatusBadRequest, "Missing paste ID")
		return
	}

	response, err := h.pasteService.GetPaste(c.Request.Context(), shortID)
	if err != nil {
		h.handleShortURLError(c, err)
		return
	}

	// Content negotiation based on Accept header
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		c.JSON(http.StatusOK, response)
		return
	}

	// Default: return plain text content
	c.Header("X-Syntax-Type", response.SyntaxType)
	c.Header("X-Created-At", response.CreatedAt)
	if response.ExpiresAt != nil {
		c.Header("X-Expires-At", *response.ExpiresAt)
	}
	c.String(http.StatusOK, response.Content)
}

// handleShortURLError handles errors for short URL endpoint (plain text responses)
func (h *PasteHandler) handleShortURLError(c *gin.Context, err error) {
	accept := c.GetHeader("Accept")
	useJSON := strings.Contains(accept, "application/json")

	switch {
	case errors.Is(err, service.ErrPasteNotFound):
		if useJSON {
			c.JSON(http.StatusNotFound, gin.H{"error": "Paste not found"})
		} else {
			c.String(http.StatusNotFound, "Paste not found")
		}
	case errors.Is(err, service.ErrPasteExpired):
		if useJSON {
			c.JSON(http.StatusGone, gin.H{"error": "Paste has expired"})
		} else {
			c.String(http.StatusGone, "Paste has expired")
		}
	default:
		if useJSON {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		} else {
			c.String(http.StatusInternalServerError, "Internal server error")
		}
	}
}

// handleError maps service errors to HTTP responses
func (h *PasteHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEmptyContent):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Content cannot be empty",
		})
	case errors.Is(err, service.ErrContentTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error":    "Content too large",
			"max_size": "1MB",
		})
	case errors.Is(err, service.ErrInvalidExpiresIn):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid expires_in value",
		})
	case errors.Is(err, service.ErrInvalidSyntaxType):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid syntax_type value",
		})
	case errors.Is(err, service.ErrNoKeysAvailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Service temporarily unavailable",
		})
	case errors.Is(err, service.ErrPasteNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Paste not found",
		})
	case errors.Is(err, service.ErrPasteExpired):
		c.JSON(http.StatusGone, gin.H{
			"error": "Paste has expired",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	}
}
