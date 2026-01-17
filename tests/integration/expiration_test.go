package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestPasteExpiration(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Paste with expiration time", func(t *testing.T) {
		// Create paste with 2 second expiration
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:   "This will expire",
			ExpiresIn: "2s",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Should be accessible immediately
		getResp, _ := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp, http.StatusOK)

		// Wait for expiration
		WaitForExpiration(2 * time.Second)

		// Should return 410 Gone after expiration
		getResp2, body := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp2, http.StatusGone)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Paste has expired" {
			t.Errorf("Expected 'Paste has expired' error, got %s", errResp.Error)
		}
	})

	t.Run("Paste without expiration (never expires)", func(t *testing.T) {
		// Create paste without expiration
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:   "This never expires",
			ExpiresIn: "never",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Check expires_at is not set
		if created.ExpiresAt != "" {
			t.Errorf("Expected expires_at to be empty, got %s", created.ExpiresAt)
		}

		// Should still be accessible
		getResp, _ := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp, http.StatusOK)
	})

	t.Run("Paste with various expiration formats", func(t *testing.T) {
		validExpirations := []string{
			"10m", "1h", "6h", "12h", "1d", "3d", "1w", "2w", "1M",
		}

		for _, exp := range validExpirations {
			t.Run(exp, func(t *testing.T) {
				createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
					Content:   "Test expiration: " + exp,
					ExpiresIn: exp,
				})
				AssertStatusCode(t, createResp, http.StatusCreated)

				created := ParseCreateResponse(t, body)
				if created.ExpiresAt == "" {
					t.Error("Expected expires_at to be set")
				}
			})
		}
	})
}

func TestBurnAfterRead(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Burn after read deletes paste on first read", func(t *testing.T) {
		// Create paste with burn after read
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:   "This will be burned after reading",
			ExpiresIn: "burn",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// First read - should succeed
		getResp, body := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp, http.StatusOK)

		paste := ParseGetResponse(t, body)
		if paste.Content != "This will be burned after reading" {
			t.Errorf("Expected content to match, got %s", paste.Content)
		}

		// Wait a bit for async deletion to complete
		time.Sleep(500 * time.Millisecond)

		// Second read - should fail (paste was burned)
		getResp2, body := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp2, http.StatusNotFound)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Paste not found" {
			t.Errorf("Expected 'Paste not found' error, got %s", errResp.Error)
		}
	})

	t.Run("Multiple burn after read pastes", func(t *testing.T) {
		// Create multiple burn-after-read pastes
		var shortIDs []string
		for i := 0; i < 3; i++ {
			createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
				Content:   "Burn paste content",
				ExpiresIn: "burn",
			})
			AssertStatusCode(t, createResp, http.StatusCreated)

			created := ParseCreateResponse(t, body)
			shortIDs = append(shortIDs, created.ShortID)
		}

		// Read each paste once (should succeed)
		for _, id := range shortIDs {
			getResp, _ := DoGetPaste(t, env.Server.URL, id)
			AssertStatusCode(t, getResp, http.StatusOK)
		}

		// Wait for async deletion
		time.Sleep(500 * time.Millisecond)

		// Try to read again (should fail)
		for _, id := range shortIDs {
			getResp, _ := DoGetPaste(t, env.Server.URL, id)
			AssertStatusCode(t, getResp, http.StatusNotFound)
		}
	})
}

func TestShortURLExpiration(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Expired paste via short URL returns 410", func(t *testing.T) {
		// Create paste with short expiration
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:   "Short URL expiration test",
			ExpiresIn: "2s",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Access via short URL (should work)
		getResp, _ := DoShortURL(t, env.Server.URL, created.ShortID, true)
		AssertStatusCode(t, getResp, http.StatusOK)

		// Wait for expiration
		WaitForExpiration(2 * time.Second)

		// Access via short URL with JSON (should return 410)
		getResp2, body := DoShortURL(t, env.Server.URL, created.ShortID, true)
		AssertStatusCode(t, getResp2, http.StatusGone)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Paste has expired" {
			t.Errorf("Expected 'Paste has expired' error, got %s", errResp.Error)
		}
	})

	t.Run("Expired paste via short URL plain text returns 410", func(t *testing.T) {
		// Create paste with short expiration
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:   "Plain text expiration test",
			ExpiresIn: "2s",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Wait for expiration
		WaitForExpiration(2 * time.Second)

		// Access via short URL without Accept header (plain text)
		getResp, body := DoShortURL(t, env.Server.URL, created.ShortID, false)
		AssertStatusCode(t, getResp, http.StatusGone)

		if string(body) != "Paste has expired" {
			t.Errorf("Expected 'Paste has expired' plain text, got %s", string(body))
		}
	})
}
