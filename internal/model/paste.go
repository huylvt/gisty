package model

import "time"

// Paste represents a paste entry in the database
type Paste struct {
	ShortID       string     `bson:"short_id" json:"short_id"`
	UserID        *string    `bson:"user_id,omitempty" json:"user_id,omitempty"`
	ContentKey    string     `bson:"content_key" json:"content_key"`
	ExpiresAt     *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt     time.Time  `bson:"created_at" json:"created_at"`
	SyntaxType    string     `bson:"syntax_type" json:"syntax_type"`
	IsPrivate     bool       `bson:"is_private" json:"is_private"`
	BurnAfterRead bool       `bson:"burn_after_read" json:"burn_after_read"`
}

// IsExpired checks if the paste has expired
func (p *Paste) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// HasExpiration returns true if the paste has an expiration time set
func (p *Paste) HasExpiration() bool {
	return p.ExpiresAt != nil
}