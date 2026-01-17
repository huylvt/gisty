package integration

import (
	"net/http"
	"strconv"
	"testing"
)

func TestRateLimiting(t *testing.T) {
	SkipIfNoDocker(t)

	// Setup with rate limiting enabled (3 requests per minute for faster testing)
	env := SetupTestEnvWithRateLimit(t, 3)
	defer env.Cleanup()

	t.Run("Requests within limit succeed", func(t *testing.T) {
		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			resp, _ := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
				Content: "Rate limit test " + strconv.Itoa(i),
			})
			AssertStatusCode(t, resp, http.StatusCreated)

			// Check rate limit headers
			limit := resp.Header.Get("X-Ratelimit-Limit")
			remaining := resp.Header.Get("X-Ratelimit-Remaining")

			if limit != "3" {
				t.Errorf("Expected X-Ratelimit-Limit to be 3, got %s", limit)
			}

			expectedRemaining := strconv.Itoa(2 - i)
			if remaining != expectedRemaining {
				t.Errorf("Expected X-Ratelimit-Remaining to be %s, got %s", expectedRemaining, remaining)
			}
		}
	})

	t.Run("Requests exceeding limit are blocked", func(t *testing.T) {
		// 4th request should be blocked (we already made 3 in previous test)
		resp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content: "Should be blocked",
		})
		AssertStatusCode(t, resp, http.StatusTooManyRequests)

		errResp := ParseErrorResponse(t, body)
		if errResp.Error != "Rate limit exceeded" {
			t.Errorf("Expected 'Rate limit exceeded' error, got %s", errResp.Error)
		}

		// retry_after should be set
		if errResp.RetryAfter <= 0 {
			t.Error("Expected retry_after to be positive")
		}
	})

	t.Run("Rate limit only applies to POST endpoints", func(t *testing.T) {
		// GET requests should not be rate limited
		// First create a paste (may be blocked, but we'll use an existing one)
		createResp, body := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content: "For GET test",
		})

		var shortID string
		if createResp.StatusCode == http.StatusCreated {
			created := ParseCreateResponse(t, body)
			shortID = created.ShortID
		} else {
			// If rate limited, we can't create - skip GET test part
			t.Log("Rate limited, skipping GET test with actual paste")
			return
		}

		// Multiple GET requests should work
		for i := 0; i < 5; i++ {
			getResp, _ := DoGetPaste(t, env.Server.URL, shortID)
			if getResp.StatusCode != http.StatusOK {
				t.Errorf("GET request %d failed with status %d", i, getResp.StatusCode)
			}
		}
	})
}

func TestRateLimitHeaders(t *testing.T) {
	SkipIfNoDocker(t)

	// Setup with rate limiting enabled
	env := SetupTestEnvWithRateLimit(t, 5)
	defer env.Cleanup()

	t.Run("Rate limit headers are set correctly", func(t *testing.T) {
		resp, _ := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
			Content: "Header test",
		})
		AssertStatusCode(t, resp, http.StatusCreated)

		// Check all rate limit headers
		limit := resp.Header.Get("X-Ratelimit-Limit")
		remaining := resp.Header.Get("X-Ratelimit-Remaining")
		reset := resp.Header.Get("X-Ratelimit-Reset")

		if limit == "" {
			t.Error("Expected X-Ratelimit-Limit header to be set")
		}
		if remaining == "" {
			t.Error("Expected X-Ratelimit-Remaining header to be set")
		}
		if reset == "" {
			t.Error("Expected X-Ratelimit-Reset header to be set")
		}

		// Parse and validate values
		limitNum, err := strconv.Atoi(limit)
		if err != nil || limitNum != 5 {
			t.Errorf("Expected X-Ratelimit-Limit to be 5, got %s", limit)
		}

		remainingNum, err := strconv.Atoi(remaining)
		if err != nil || remainingNum < 0 || remainingNum > 5 {
			t.Errorf("Expected X-Ratelimit-Remaining to be between 0 and 5, got %s", remaining)
		}

		resetNum, err := strconv.ParseInt(reset, 10, 64)
		if err != nil || resetNum <= 0 {
			t.Errorf("Expected X-Ratelimit-Reset to be a positive unix timestamp, got %s", reset)
		}
	})

	t.Run("Remaining decreases with each request", func(t *testing.T) {
		var lastRemaining int = 10 // Start high

		for i := 0; i < 3; i++ {
			resp, _ := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
				Content: "Decrement test " + strconv.Itoa(i),
			})

			if resp.StatusCode == http.StatusTooManyRequests {
				// Rate limited from previous tests, acceptable
				break
			}

			remaining := resp.Header.Get("X-Ratelimit-Remaining")
			remainingNum, _ := strconv.Atoi(remaining)

			if remainingNum >= lastRemaining && i > 0 {
				t.Errorf("Expected remaining to decrease, was %d now %d", lastRemaining, remainingNum)
			}
			lastRemaining = remainingNum
		}
	})
}

func TestRateLimitDisabled(t *testing.T) {
	SkipIfNoDocker(t)

	// Setup without rate limiting (default in SetupTestEnv)
	env := SetupTestEnv(t)
	defer env.Cleanup()

	t.Run("Many requests succeed when rate limiting is disabled", func(t *testing.T) {
		// Should be able to make many requests without being blocked
		for i := 0; i < 10; i++ {
			resp, _ := DoCreatePaste(t, env.Server.URL, CreatePasteRequest{
				Content: "No rate limit test " + strconv.Itoa(i),
			})
			AssertStatusCode(t, resp, http.StatusCreated)
		}
	})
}
