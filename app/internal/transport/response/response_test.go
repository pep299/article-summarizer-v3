package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	resp := Response{
		Status:  "success",
		Message: "test message",
	}

	err := WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var result Response
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}

	if result.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", result.Message)
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteError(w, http.StatusInternalServerError, "test error")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var result Response
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error != "test error" {
		t.Errorf("Expected error 'test error', got '%s'", result.Error)
	}
}

func TestWriteSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	err := WriteSuccess(w, "operation successful", data)
	if err != nil {
		t.Fatalf("WriteSuccess failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result Response
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", result.Status)
	}

	if result.Message != "operation successful" {
		t.Errorf("Expected message 'operation successful', got '%s'", result.Message)
	}

	// Check data field
	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Error("Expected data to be a map")
	} else if dataMap["key"] != "value" {
		t.Errorf("Expected data.key 'value', got '%v'", dataMap["key"])
	}
}

func TestWriteBadRequest(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteBadRequest(w, "invalid input")
	if err != nil {
		t.Fatalf("WriteBadRequest failed: %v", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var result Response
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error != "invalid input" {
		t.Errorf("Expected error 'invalid input', got '%s'", result.Error)
	}
}

func TestWriteInternalError(t *testing.T) {
	w := httptest.NewRecorder()

	err := WriteInternalError(w, "internal server error")
	if err != nil {
		t.Fatalf("WriteInternalError failed: %v", err)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var result Response
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", result.Status)
	}

	if result.Error != "internal server error" {
		t.Errorf("Expected error 'internal server error', got '%s'", result.Error)
	}
}
