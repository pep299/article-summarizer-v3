package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockHandler is a simple handler for testing
func mockHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
}

// TestAuth_ValidRequest tests authentication with valid token
func TestAuth_ValidRequest(t *testing.T) {
	token := "test-secret-token"
	authMiddleware := Auth(token)
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	t.Logf("Request: %s %s", req.Method, req.URL.Path)
	t.Logf("Auth header: %s", req.Header.Get("Authorization"))
	t.Logf("Response status: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "success" {
		t.Errorf("Expected 'success', got '%s'", w.Body.String())
	}
}

// TestAuth_InvalidToken tests authentication with invalid token
func TestAuth_InvalidToken(t *testing.T) {
	token := "test-secret-token"
	authMiddleware := Auth(token)
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	req.Header.Set("Authorization", "Bearer wrong-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestAuth_MissingToken tests authentication with missing Authorization header
func TestAuth_MissingToken(t *testing.T) {
	token := "test-secret-token"
	authMiddleware := Auth(token)
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestAuth_WrongBearerFormat tests authentication with wrong Bearer format
func TestAuth_WrongBearerFormat(t *testing.T) {
	token := "test-secret-token"
	authMiddleware := Auth(token)
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	req.Header.Set("Authorization", "Basic "+token) // Wrong format
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

// TestAuth_NonPOSTMethod tests that non-POST methods are rejected
func TestAuth_NonPOSTMethod(t *testing.T) {
	token := "test-secret-token"
	authMiddleware := Auth(token)
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	// Test GET method
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestAuth_EmptyToken tests authentication middleware with empty token
func TestAuth_EmptyToken(t *testing.T) {
	authMiddleware := Auth("")
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	// Even with empty token configured, Bearer header should match
	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	req.Header.Set("Authorization", "Bearer ")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 with empty token match, got %d", w.Code)
	}
}

// TestAuth_PartialTokenMatch tests that partial token matches are rejected
func TestAuth_PartialTokenMatch(t *testing.T) {
	token := "test-secret-token"
	authMiddleware := Auth(token)
	handler := authMiddleware(http.HandlerFunc(mockHandler))

	req := httptest.NewRequest("POST", "/test", strings.NewReader("test body"))
	req.Header.Set("Authorization", "Bearer test-secret") // Partial match
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for partial token match, got %d", w.Code)
	}
}
