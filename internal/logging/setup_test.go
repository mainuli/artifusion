package logging

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected zerolog.Level
	}{
		{
			name:     "debug level",
			input:    "debug",
			expected: zerolog.DebugLevel,
		},
		{
			name:     "info level",
			input:    "info",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "warn level",
			input:    "warn",
			expected: zerolog.WarnLevel,
		},
		{
			name:     "error level",
			input:    "error",
			expected: zerolog.ErrorLevel,
		},
		{
			name:     "unknown level defaults to info",
			input:    "invalid",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "empty level defaults to info",
			input:    "",
			expected: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewLogger_JSON(t *testing.T) {
	// Set a known starting state
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	logger := NewLogger(
		Config{
			Level:  "info",
			Format: "json",
		},
		"test-service",
		"1.0.0",
	)

	// Verify logger is created successfully
	// Zerolog uses global level filtering, so we just verify logger was created
	_ = logger

	// Verify global level was set correctly
	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Errorf("expected global level info, got %v", zerolog.GlobalLevel())
	}
}

func TestNewLogger_Console(t *testing.T) {
	logger := NewLogger(
		Config{
			Level:  "debug",
			Format: "console",
		},
		"test-service",
		"1.0.0",
	)

	// Verify logger is created successfully
	_ = logger

	// Verify global level was set correctly
	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Errorf("expected global level debug, got %v", zerolog.GlobalLevel())
	}
}

func TestNewLogger_DifferentLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
	}{
		{"debug", "debug", zerolog.DebugLevel},
		{"info", "info", zerolog.InfoLevel},
		{"warn", "warn", zerolog.WarnLevel},
		{"error", "error", zerolog.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewLogger(
				Config{
					Level:  tt.level,
					Format: "json",
				},
				"test-service",
				"1.0.0",
			)

			// Verify global level was set correctly by NewLogger
			if zerolog.GlobalLevel() != tt.expected {
				t.Errorf("expected global level %v, got %v", tt.expected, zerolog.GlobalLevel())
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	// Test with stdout
	// Note: This test may return false in CI/CD environments where stdout is not a TTY
	result := isTerminal(os.Stdout)
	t.Logf("isTerminal(os.Stdout) = %v", result)

	// Create a temporary file (not a terminal)
	tmpFile, err := os.CreateTemp("", "test-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("failed to remove temp file: %v", err)
		}
	}()
	defer func() {
		if err := tmpFile.Close(); err != nil {
			t.Logf("failed to close temp file: %v", err)
		}
	}()

	// File should not be detected as a terminal
	if isTerminal(tmpFile) {
		t.Error("expected false for regular file, got true")
	}
}

func TestNewConsoleLogger(t *testing.T) {
	// Just verify it doesn't panic and returns a valid logger
	logger := newConsoleLogger()
	_ = logger // Logger created successfully
}

func TestNewJSONLogger(t *testing.T) {
	// Just verify it doesn't panic and returns a valid logger
	logger := newJSONLogger("test-service", "1.0.0")
	_ = logger // Logger created successfully
}
