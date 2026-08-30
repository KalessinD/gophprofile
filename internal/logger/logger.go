//go:generate mockgen -source=logger.go -destination=mocks/mock_logger.gen.go -package=mocks
package logger

import (
	"fmt"
)

// Logger engine constants for type-safe configuration.
const (
	EngineSlog = "slog"
	EngineZap  = "zap"
	EngineNop  = "nop"
)

// Logger defines the universal contract for logging.
// It supports both structured (key-value) and formatted logging,
// allowing to switch between underlying implementations (slog, zap, etc.).
type Logger interface {
	// Sugar returns the logger itself, as all methods in this interface
	// already follow the "sugared" pattern (accepting variadic keys or format args).
	Sugar() Logger

	// With creates a child logger with additional structured context fields.
	With(keysAndValues ...any) Logger

	// Structured logging methods (key-value pairs)
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)

	// Formatted logging methods (fmt.Sprintf style)
	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)

	// Sync flushes any buffered log entries. Should be called before app exit.
	Sync() error
}

// NewLogger is a unified factory that returns a Logger interface based on the chosen engine.
// If an empty string is provided, it defaults to "slog".
func NewLogger(engine string, isProd bool) (Logger, error) {
	if engine == "" {
		engine = EngineSlog
	}

	switch engine {
	case EngineSlog:
		return NewSlogLogger(isProd), nil
	case EngineZap:
		return NewZapLogger(isProd)
	case EngineNop:
		return NewNopLogger(), nil
	default:
		return nil, fmt.Errorf("unsupported logger engine: %s", engine)
	}
}
