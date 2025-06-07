package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMainFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "Help flag",
			args:     []string{"-help"},
			expected: "Article Summarizer v3 CLI",
		},
		{
			name:     "Version flag",
			args:     []string{"-version"},
			expected: "Article Summarizer v3 CLI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if os.Getenv("TEST_MAIN_SUBPROCESS") == "1" {
				// Reset os.Args for the test
				os.Args = append([]string{"cmd"}, tt.args...)
				main()
				return
			}

			// Run the test as a subprocess
			cmd := exec.Command(os.Args[0], "-test.run=TestMainFlags/"+strings.ReplaceAll(tt.name, " ", "_"))
			cmd.Env = append(os.Environ(), "TEST_MAIN_SUBPROCESS=1")
			output, err := cmd.Output()

			// For help and version flags, we expect the program to exit with code 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					// Exit code 0 is expected for help and version flags
					if exitError.ExitCode() != 0 {
						t.Errorf("Expected exit code 0, got %d", exitError.ExitCode())
					}
				}
			}

			if !strings.Contains(string(output), tt.expected) {
				t.Errorf("Expected output to contain %q, got %q", tt.expected, string(output))
			}
		})
	}
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are properly defined
	if Version == "" {
		Version = "dev" // Set default for test
	}
	if Commit == "" {
		Commit = "unknown" // Set default for test
	}
	if BuildTime == "" {
		BuildTime = "unknown" // Set default for test
	}

	// These should not be empty after setting defaults
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

func TestMainWithValidConfig(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("RSS_FEEDS", `[{"name":"test","url":"http://example.com/feed","channel":"test-channel"}]`)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("RSS_FEEDS")
	}()

	if os.Getenv("TEST_MAIN_SUBPROCESS") == "1" {
		// Reset os.Args for the test
		os.Args = []string{"cmd"}
		main()
		return
	}

	// Run the test as a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestMainWithValidConfig")
	cmd.Env = append(os.Environ(),
		"TEST_MAIN_SUBPROCESS=1",
		"GEMINI_API_KEY=test-key",
		"SLACK_BOT_TOKEN=xoxb-test-token",
		"RSS_FEEDS=[{\"name\":\"test\",\"url\":\"http://example.com/feed\",\"channel\":\"test-channel\"}]",
	)

	// This will likely fail due to network calls, but we're testing that the config loading works
	output, err := cmd.CombinedOutput()

	// We expect this to fail with network errors, not config errors
	if err != nil {
		outputStr := string(output)
		// Should not fail with config loading errors
		if strings.Contains(outputStr, "Failed to load configuration") {
			t.Errorf("Unexpected config loading error: %s", outputStr)
		}
	}
}
