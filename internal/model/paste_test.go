package model

import (
	"testing"
	"time"
)

func TestPaste_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name: "expired",
			expiresAt: func() *time.Time {
				t := time.Now().Add(-1 * time.Hour)
				return &t
			}(),
			want: true,
		},
		{
			name: "not expired",
			expiresAt: func() *time.Time {
				t := time.Now().Add(1 * time.Hour)
				return &t
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Paste{
				ExpiresAt: tt.expiresAt,
			}
			if got := p.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaste_HasExpiration(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "no expiration",
			expiresAt: nil,
			want:      false,
		},
		{
			name: "has expiration",
			expiresAt: func() *time.Time {
				t := time.Now().Add(1 * time.Hour)
				return &t
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Paste{
				ExpiresAt: tt.expiresAt,
			}
			if got := p.HasExpiration(); got != tt.want {
				t.Errorf("HasExpiration() = %v, want %v", got, tt.want)
			}
		})
	}
}