package adapters

import (
	"testing"
)

func TestNoOpLoggerAdapter(t *testing.T) {
	logger := NewNoOpLoggerAdapter()

	// Test all methods - they should not panic and do nothing
	logger.Debug("debug message", "arg1", "arg2")
	logger.Info("info message", "arg1", "arg2")
	logger.Warn("warn message", "arg1", "arg2")
	logger.Error("error message", "arg1", "arg2")

	// If we reach here without panic, the test passes
}

func TestNoOpLoggerAdapter_AllMethods(t *testing.T) {
	logger := NewNoOpLoggerAdapter()

	// Call each method individually to ensure coverage
	t.Run("Debug", func(t *testing.T) {
		logger.Debug("test")
	})

	t.Run("Info", func(t *testing.T) {
		logger.Info("test")
	})

	t.Run("Warn", func(t *testing.T) {
		logger.Warn("test")
	})

	t.Run("Error", func(t *testing.T) {
		logger.Error("test")
	})
}

func TestNoOpLoggerAdapter_WithVariousArgs(t *testing.T) {
	logger := NewNoOpLoggerAdapter()

	// Test with no arguments
	logger.Debug("message")
	logger.Info("message")
	logger.Warn("message")
	logger.Error("message")

	// Test with multiple arguments
	logger.Debug("message %s %d", "test", 123)
	logger.Info("message %s %d", "test", 123)
	logger.Warn("message %s %d", "test", 123)
	logger.Error("message %s %d", "test", 123)

	// Test with nil arguments
	logger.Debug("message", nil)
	logger.Info("message", nil)
	logger.Warn("message", nil)
	logger.Error("message", nil)
}

func TestNoOpLoggerAdapter_Constructor(t *testing.T) {
	logger := NewNoOpLoggerAdapter()
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}

	// Verify it implements LoggerAdapter interface
	var _ LoggerAdapter = logger
}
