package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger implements Logger interface using go.uber.org/zap.
type ZapLogger struct {
	sugar *zap.SugaredLogger
}

// NewZapLogger creates a new Logger instance backed by Zap.
// It replaces the old NewLogger function.
func NewZapLogger(isProd bool) (Logger, error) {
	var baseLogger *zap.Logger
	var err error

	if isProd {
		baseLogger, err = zap.NewProduction()
	} else {
		baseLogger, err = newConsoleLogger()
	}

	if err != nil {
		return nil, err
	}

	return &ZapLogger{sugar: baseLogger.Sugar()}, nil
}

// newConsoleLogger creates a development zap logger with color levels.
func newConsoleLogger() (*zap.Logger, error) {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.DebugLevel,
	)

	return zap.New(core, zap.AddCaller()), nil
}

// Sugar returns the logger itself, as ZapLogger already uses the SugaredLogger API.
func (zl *ZapLogger) Sugar() Logger {
	return zl
}

// With creates a child logger with additional structured context fields.
func (zl *ZapLogger) With(keysAndValues ...any) Logger {
	return &ZapLogger{sugar: zl.sugar.With(keysAndValues...)}
}

// Debug logs a message at DebugLevel with structured key-value pairs.
func (zl *ZapLogger) Debug(msg string, keysAndValues ...any) {
	zl.sugar.Debugw(msg, keysAndValues...)
}

// Info logs a message at InfoLevel with structured key-value pairs.
func (zl *ZapLogger) Info(msg string, keysAndValues ...any) {
	zl.sugar.Infow(msg, keysAndValues...)
}

// Warn logs a message at WarnLevel with structured key-value pairs.
func (zl *ZapLogger) Warn(msg string, keysAndValues ...any) {
	zl.sugar.Warnw(msg, keysAndValues...)
}

// Error logs a message at ErrorLevel with structured key-value pairs.
func (zl *ZapLogger) Error(msg string, keysAndValues ...any) {
	zl.sugar.Errorw(msg, keysAndValues...)
}

// Debugf logs a message at DebugLevel using fmt.Sprintf formatting.
func (zl *ZapLogger) Debugf(template string, args ...any) {
	zl.sugar.Debugf(template, args...)
}

// Infof logs a message at InfoLevel using fmt.Sprintf formatting.
func (zl *ZapLogger) Infof(template string, args ...any) {
	zl.sugar.Infof(template, args...)
}

// Warnf logs a message at WarnLevel using fmt.Sprintf formatting.
func (zl *ZapLogger) Warnf(template string, args ...any) {
	zl.sugar.Warnf(template, args...)
}

// Errorf logs a message at ErrorLevel using fmt.Sprintf formatting.
func (zl *ZapLogger) Errorf(template string, args ...any) {
	zl.sugar.Errorf(template, args...)
}

// Sync flushes any buffered log entries.
func (zl *ZapLogger) Sync() error {
	return zl.sugar.Sync()
}
