package adapters

import (
	"log"
)

// PrintLoggerAdapter implements LoggerAdapter using standard log package
type PrintLoggerAdapter struct {
	level LogLevel
}

// NewPrintLoggerAdapter creates a new print logger with the specified level
func NewPrintLoggerAdapter(level LogLevel) *PrintLoggerAdapter {
	return &PrintLoggerAdapter{level: level}
}

func (p *PrintLoggerAdapter) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelNone:  4,
	}
	return levels[level] >= levels[p.level]
}

func (p *PrintLoggerAdapter) Debug(message string, args ...interface{}) {
	if p.shouldLog(LogLevelDebug) {
		log.Printf("[DEBUG] [Ripple] "+message, args...)
	}
}

func (p *PrintLoggerAdapter) Info(message string, args ...interface{}) {
	if p.shouldLog(LogLevelInfo) {
		log.Printf("[INFO] [Ripple] "+message, args...)
	}
}

func (p *PrintLoggerAdapter) Warn(message string, args ...interface{}) {
	if p.shouldLog(LogLevelWarn) {
		log.Printf("[WARN] [Ripple] "+message, args...)
	}
}

func (p *PrintLoggerAdapter) Error(message string, args ...interface{}) {
	if p.shouldLog(LogLevelError) {
		log.Printf("[ERROR] [Ripple] "+message, args...)
	}
}
