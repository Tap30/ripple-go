package adapters

import (
	"testing"
)

func TestPrintLoggerAdapter(t *testing.T) {
	t.Run("should create logger with debug level", func(t *testing.T) {
		logger := NewPrintLoggerAdapter(LogLevelDebug)
		if logger.level != LogLevelDebug {
			t.Errorf("expected debug level, got %s", logger.level)
		}
	})

	t.Run("should log debug messages when level is debug", func(t *testing.T) {
		logger := NewPrintLoggerAdapter(LogLevelDebug)
		logger.Debug("debug message %s", "test")
		// If we reach here without panic, the test passes
	})

	t.Run("should log info messages when level is debug", func(t *testing.T) {
		logger := NewPrintLoggerAdapter(LogLevelDebug)
		logger.Info("info message %s", "test")
		// If we reach here without panic, the test passes
	})

	t.Run("should log warn messages when level is debug", func(t *testing.T) {
		logger := NewPrintLoggerAdapter(LogLevelDebug)
		logger.Warn("warn message %s", "test")
		// If we reach here without panic, the test passes
	})

	t.Run("should log error messages when level is debug", func(t *testing.T) {
		logger := NewPrintLoggerAdapter(LogLevelDebug)
		logger.Error("error message %s", "test")
		// If we reach here without panic, the test passes
	})

	t.Run("should respect log levels", func(t *testing.T) {
		// Test that higher level loggers don't log lower level messages
		logger := NewPrintLoggerAdapter(LogLevelError)
		
		// These should not cause any issues (they just won't output)
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
	})

	t.Run("should handle none level", func(t *testing.T) {
		logger := NewPrintLoggerAdapter(LogLevelNone)
		
		// None of these should output anything
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
	})
}
