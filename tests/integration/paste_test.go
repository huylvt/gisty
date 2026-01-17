package integration

import (
	"net/http"
	"testing"
)

func TestPasteCRUD(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Create paste", func(t *testing.T) {
		resp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:    "Hello, World!",
			SyntaxType: "text",
		})

		AssertStatusCode(t, resp, http.StatusCreated)

		createResp := ParseCreateResponse(t, body)
		if createResp.ShortID == "" {
			t.Error("Expected short_id to be set")
		}
		if createResp.URL == "" {
			t.Error("Expected url to be set")
		}
	})

	t.Run("Create and Get paste", func(t *testing.T) {
		// Create
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:    "def hello():\n    print('Hello')",
			SyntaxType: "python",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Get
		getResp, body := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp, http.StatusOK)

		paste := ParseGetResponse(t, body)
		if paste.Content != "def hello():\n    print('Hello')" {
			t.Errorf("Expected content to match, got %s", paste.Content)
		}
		if paste.SyntaxType != "python" {
			t.Errorf("Expected syntax_type to be python, got %s", paste.SyntaxType)
		}
		if paste.ShortID != created.ShortID {
			t.Errorf("Expected short_id to match, got %s", paste.ShortID)
		}
	})

	t.Run("Create, Get, and Delete paste", func(t *testing.T) {
		// Create
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content: "To be deleted",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Get (should exist)
		getResp, _ := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp, http.StatusOK)

		// Delete
		deleteResp, _ := DoDeletePaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, deleteResp, http.StatusNoContent)

		// Get (should not exist)
		getResp2, body := DoGetPaste(t, env.Server.URL, created.ShortID)
		AssertStatusCode(t, getResp2, http.StatusNotFound)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Paste not found" {
			t.Errorf("Expected 'Paste not found' error, got %s", errResp.Error)
		}
	})

	t.Run("Get non-existent paste", func(t *testing.T) {
		resp, body := DoGetPaste(t, env.Server.URL, "nonexistent123")
		AssertStatusCode(t, resp, http.StatusNotFound)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Paste not found" {
			t.Errorf("Expected 'Paste not found' error, got %s", errResp.Error)
		}
	})

	t.Run("Delete non-existent paste", func(t *testing.T) {
		resp, body := DoDeletePaste(t, env.Server.URL, "nonexistent123")
		AssertStatusCode(t, resp, http.StatusNotFound)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Paste not found" {
			t.Errorf("Expected 'Paste not found' error, got %s", errResp.Error)
		}
	})
}

func TestPasteShortURL(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Short URL returns JSON with Accept header", func(t *testing.T) {
		// Create
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:    "Short URL test",
			SyntaxType: "text",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Get via short URL with Accept: application/json
		getResp, body := DoShortURL(t, env.Server.URL, created.ShortID, true)
		AssertStatusCode(t, getResp, http.StatusOK)

		paste := ParseGetResponse(t, body)
		if paste.Content != "Short URL test" {
			t.Errorf("Expected content to match, got %s", paste.Content)
		}
	})

	t.Run("Short URL returns plain text without Accept header", func(t *testing.T) {
		// Create
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:    "Plain text response",
			SyntaxType: "text",
		})
		AssertStatusCode(t, createResp, http.StatusCreated)

		created := ParseCreateResponse(t, body)

		// Get via short URL without Accept header
		getResp, body := DoShortURL(t, env.Server.URL, created.ShortID, false)
		AssertStatusCode(t, getResp, http.StatusOK)

		// Should return plain text
		if string(body) != "Plain text response" {
			t.Errorf("Expected plain text 'Plain text response', got %s", string(body))
		}

		// Check headers
		if getResp.Header.Get("X-Syntax-Type") != "text" {
			t.Errorf("Expected X-Syntax-Type header to be 'text', got %s", getResp.Header.Get("X-Syntax-Type"))
		}
		if getResp.Header.Get("X-Created-At") == "" {
			t.Error("Expected X-Created-At header to be set")
		}
	})
}

func TestPasteAutoDetectLanguage(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	tests := []struct {
		name           string
		content        string
		expectedSyntax string
	}{
		{
			name:           "Python code",
			content:        "def hello():\n    print('Hello, World!')\n\nif __name__ == '__main__':\n    hello()",
			expectedSyntax: "python",
		},
		{
			name:           "Go code",
			content:        "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}",
			expectedSyntax: "go",
		},
		{
			name:           "JSON",
			content:        `{"name": "John", "age": 30}`,
			expectedSyntax: "json",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create without syntax_type
			createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
				Content: tc.content,
				// SyntaxType is empty - should auto-detect
			})
			AssertStatusCode(t, createResp, http.StatusCreated)

			created := ParseCreateResponse(t, body)

			// Get and check syntax type
			getResp, body := DoGetPaste(t, env.Server.URL, created.ShortID)
			AssertStatusCode(t, getResp, http.StatusOK)

			paste := ParseGetResponse(t, body)
			if paste.SyntaxType != tc.expectedSyntax {
				t.Errorf("Expected syntax_type to be %s, got %s", tc.expectedSyntax, paste.SyntaxType)
			}
		})
	}
}

func TestPasteValidation(t *testing.T) {
	SkipIfNoDocker(t)

	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Empty content", func(t *testing.T) {
		resp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content: "",
		})
		AssertStatusCode(t, resp, http.StatusBadRequest)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Invalid request body" {
			t.Errorf("Expected 'Invalid request body' error, got %s", errResp.Error)
		}
	})

	t.Run("Invalid syntax_type", func(t *testing.T) {
		resp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:    "Hello",
			SyntaxType: "invalid_language",
		})
		AssertStatusCode(t, resp, http.StatusBadRequest)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Invalid syntax_type value" {
			t.Errorf("Expected 'Invalid syntax_type value' error, got %s", errResp.Error)
		}
	})

	t.Run("Invalid expires_in", func(t *testing.T) {
		resp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content:   "Hello",
			ExpiresIn: "invalid_duration",
		})
		AssertStatusCode(t, resp, http.StatusBadRequest)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Invalid expires_in value" {
			t.Errorf("Expected 'Invalid expires_in value' error, got %s", errResp.Error)
		}
	})
}
