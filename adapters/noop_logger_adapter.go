package adapters

// NoOpLoggerAdapter implements LoggerAdapter with no-op methods
type NoOpLoggerAdapter struct{}

// NewNoOpLoggerAdapter creates a new no-op logger
func NewNoOpLoggerAdapter() *NoOpLoggerAdapter {
	return &NoOpLoggerAdapter{}
}

func (n *NoOpLoggerAdapter) Debug(message string, args ...any) {}
func (n *NoOpLoggerAdapter) Info(message string, args ...any)  {}
func (n *NoOpLoggerAdapter) Warn(message string, args ...any)  {}
func (n *NoOpLoggerAdapter) Error(message string, args ...any) {}
