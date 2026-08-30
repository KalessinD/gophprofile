package logger

// NopLogger implements Logger interface and discards all log messages.
// Useful for testing.
type NopLogger struct{}

// NewNopLogger creates a new no-operation Logger instance.
func NewNopLogger() Logger {
	return &NopLogger{}
}

// Sugar returns the logger itself.
func (n *NopLogger) Sugar() Logger {
	return n
}

// With returns a new NopLogger (ignores context fields).
func (n *NopLogger) With(_ ...any) Logger {
	return &NopLogger{}
}

// Debug is a no-op.
func (n *NopLogger) Debug(_ string, _ ...any) {}

// Info is a no-op.
func (n *NopLogger) Info(_ string, _ ...any) {}

// Warn is a no-op.
func (n *NopLogger) Warn(_ string, _ ...any) {}

// Error is a no-op.
func (n *NopLogger) Error(_ string, _ ...any) {}

// Debugf is a no-op.
func (n *NopLogger) Debugf(_ string, _ ...any) {}

// Infof is a no-op.
func (n *NopLogger) Infof(_ string, _ ...any) {}

// Warnf is a no-op.
func (n *NopLogger) Warnf(_ string, _ ...any) {}

// Errorf is a no-op.
func (n *NopLogger) Errorf(_ string, _ ...any) {}

// Sync is a no-op.
func (n *NopLogger) Sync() error {
	return nil
}
