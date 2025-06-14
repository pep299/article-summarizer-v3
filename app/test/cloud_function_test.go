package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// This tests the complete flow: cloud_function.go init() -> funcframework.Start() -> HTTP server.
func TestMain_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment for the server process
	env := []string{
		"FUNCTION_TARGET=ProcessArticle",
		"GEMINI_API_KEY=test-key",
		"SLACK_BOT_TOKEN=xoxb-test-token",
		"SLACK_CHANNEL=#test",
		"PORT=0", // Let the system choose a free port
	}

	// Start the server process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "cmd/server/main.go")
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = ".." // Run from app directory

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			cmd.Wait()
		}
	}()

	// Wait for server to start (funcframework default port is 8080)
	serverURL := "http://localhost:8080"
	t.Logf("Waiting for server to start at %s", serverURL)
	if !waitForServer(t, serverURL, 10*time.Second) {
		t.Fatal("Server did not start within timeout")
	}
	t.Logf("Server is ready")

	// Test health check endpoint
	resp, err := http.Get(serverURL + "/hc")
	if err != nil {
		t.Fatalf("Failed to make request to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}

	t.Logf("✅ Main integration test passed - server startup and health check working")
}

// waitForServer waits for the server to become available.
func waitForServer(t *testing.T, url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/hc")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// TestMain_EnvValidation tests that missing FUNCTION_TARGET causes server startup failure.
func TestMain_EnvValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup environment WITHOUT FUNCTION_TARGET
	env := []string{
		"GEMINI_API_KEY=test-key",
		"SLACK_BOT_TOKEN=xoxb-test-token",
		"SLACK_CHANNEL=#test",
	}

	// Try to start the server process
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "cmd/server/main.go")
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = ".." // Run from app directory

	// This should fail due to missing FUNCTION_TARGET
	err := cmd.Run()
	if err == nil {
		t.Error("Expected server to fail without FUNCTION_TARGET, but it succeeded")
	}

	// Check if it's the expected error (process exited)
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 0 {
			t.Logf("✅ Server correctly failed to start without FUNCTION_TARGET (exit code: %d)", exitErr.ExitCode())
		}
	} else {
		t.Logf("✅ Server failed to start as expected: %v", err)
	}
}
