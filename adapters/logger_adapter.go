package adapters

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelNone  LogLevel = "NONE"
)

// LoggerAdapter is an interface for logging.
// Implement this interface to use custom loggers.
type LoggerAdapter interface {
	// Debug logs a debug message
	Debug(message string, args ...any)
	// Info logs an info message
	Info(message string, args ...any)
	// Warn logs a warning message
	Warn(message string, args ...any)
	// Error logs an error message
	Error(message string, args ...any)
}
