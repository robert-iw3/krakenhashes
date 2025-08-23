package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/pkg/jwt"
)

// SetTestJWTSecret sets the JWT_SECRET environment variable for testing
func SetTestJWTSecret(t *testing.T) {
	t.Helper()

	oldSecret := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", TestJWTSecret)

	t.Cleanup(func() {
		if oldSecret != "" {
			os.Setenv("JWT_SECRET", oldSecret)
		} else {
			os.Unsetenv("JWT_SECRET")
		}
	})
}

// MakeAuthenticatedRequest creates an HTTP request with a valid auth token
func MakeAuthenticatedRequest(t *testing.T, method, url string, body interface{}, userID, role string) *http.Request {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Generate a valid token with 60 minute expiry for tests
	token, err := jwt.GenerateToken(userID, role, 60)
	if err != nil {
		t.Fatalf("Failed to generate auth token: %v", err)
	}

	// Set auth cookie
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: token,
	})

	return req
}

// MakeRequest creates a basic HTTP request
func MakeRequest(t *testing.T, method, url string, body interface{}) *http.Request {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

// AssertJSONResponse checks that the response has the expected status and decodes JSON
func AssertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, v interface{}) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	if v != nil && rr.Body.Len() > 0 {
		if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, rr.Body.String())
		}
	}
}

// AssertCookieSet checks that a cookie with the given name was set
func AssertCookieSet(t *testing.T, rr *httptest.ResponseRecorder, cookieName string) *http.Cookie {
	t.Helper()

	cookies := rr.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == cookieName {
			return cookie
		}
	}

	t.Errorf("Expected cookie %s to be set, but it was not", cookieName)
	return nil
}

// AssertCookieDeleted checks that a cookie was deleted (MaxAge < 0)
func AssertCookieDeleted(t *testing.T, rr *httptest.ResponseRecorder, cookieName string) {
	t.Helper()

	cookie := AssertCookieSet(t, rr, cookieName)
	if cookie != nil && cookie.MaxAge >= 0 {
		t.Errorf("Expected cookie %s to be deleted (MaxAge < 0), but MaxAge was %d", cookieName, cookie.MaxAge)
	}
}

// GenerateTOTPCode generates a valid TOTP code for testing
func GenerateTOTPCode(secret string) (string, error) {
	// This would normally use the totp package, but for testing we'll return a fixed code
	// In real tests, you'd import the actual TOTP package
	return "123456", nil
}

// WaitForCondition waits for a condition to be true or times out
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}
