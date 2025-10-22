package logging

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
)

// Config contains logging configuration
type Config struct {
	Level      string
	Format     string
	ForceColor bool
}

// NewLogger creates a configured zerolog logger based on the provided configuration
func NewLogger(cfg Config, service, version string) zerolog.Logger {
	// Parse and set log level
	level := parseLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure output format
	if cfg.Format == "console" {
		return newConsoleLogger(cfg)
	}

	// JSON output for production
	return newJSONLogger(service, version)
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

// newConsoleLogger creates a colorful console logger for development
func newConsoleLogger(cfg Config) zerolog.Logger {
	// Auto-detect terminal color support, unless ForceColor is enabled
	noColor := !cfg.ForceColor && !isTerminal(os.Stdout)

	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02T15:04:05.000Z07:00", // ISO8601 format with milliseconds
		NoColor:    noColor,
		// Format: [TIME] LEVEL message key=value
		FormatLevel: func(i interface{}) string {
			var level string
			var color string

			if ll, ok := i.(string); ok {
				switch ll {
				case "debug":
					level = "DBG"
					color = "\033[35m" // Magenta
				case "info":
					level = "INF"
					color = "\033[36m" // Cyan
				case "warn":
					level = "WRN"
					color = "\033[33m" // Yellow
				case "error":
					level = "ERR"
					color = "\033[31m" // Red
				case "fatal":
					level = "FTL"
					color = "\033[31m\033[1m" // Bold Red
				case "panic":
					level = "PNC"
					color = "\033[31m\033[1m" // Bold Red
				default:
					level = "???"
					color = "\033[37m" // White
				}
			}

			if noColor {
				return fmt.Sprintf("| %-3s |", level)
			}
			return fmt.Sprintf("%s| %-3s |\033[0m", color, level)
		},
		FormatMessage: func(i interface{}) string {
			if i == nil {
				return ""
			}
			return fmt.Sprintf("%s", i)
		},
		FormatFieldName: func(i interface{}) string {
			return fmt.Sprintf("%s=", i)
		},
		FormatFieldValue: func(i interface{}) string {
			return fmt.Sprintf("%s", i)
		},
		// Don't show the full file path - it's too verbose for console
		PartsExclude: []string{zerolog.CallerFieldName},
	}

	return zerolog.New(output).
		With().
		Timestamp().
		Logger()
}

// newJSONLogger creates a structured JSON logger for production
func newJSONLogger(service, version string) zerolog.Logger {
	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", service).
		Str("version", version).
		Logger()
}

// isTerminal checks if the file descriptor is a terminal (supports colors)
func isTerminal(f *os.File) bool {
	// Check if stdout is a terminal (TTY)
	// This works on Unix-like systems (Linux, macOS)
	stat, err := f.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	// ModeCharDevice is set for terminals
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return true
	}

	return false
}
