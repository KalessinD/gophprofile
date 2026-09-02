package logger_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KalessinD/gophprofile/internal/logger"
)

// captureStdout captures anything written to os.Stdout during the execution of fn.
// Note: Loggers must be initialized INSIDE fn, because they capture os.Stdout at creation time.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w

	fn()

	err = w.Close()
	require.NoError(t, err)
	os.Stdout = old

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String()
}

func TestNewLogger(t *testing.T) {
	t.Run("returns slog logger by default when empty string is provided", func(t *testing.T) {
		l, err := logger.NewLogger("", false)
		require.NoError(t, err)
		assert.IsType(t, &logger.SlogLogger{}, l)
	})

	t.Run("returns slog logger explicitly", func(t *testing.T) {
		l, err := logger.NewLogger(logger.EngineSlog, false)
		require.NoError(t, err)
		assert.IsType(t, &logger.SlogLogger{}, l)
	})

	t.Run("returns zap logger", func(t *testing.T) {
		l, err := logger.NewLogger(logger.EngineZap, false)
		require.NoError(t, err)
		assert.IsType(t, &logger.ZapLogger{}, l)
	})

	t.Run("returns nop logger", func(t *testing.T) {
		l, err := logger.NewLogger(logger.EngineNop, false)
		require.NoError(t, err)
		assert.IsType(t, &logger.NopLogger{}, l)
	})

	t.Run("returns error for unsupported engine", func(t *testing.T) {
		l, err := logger.NewLogger("invalid_engine", false)
		require.Error(t, err)
		assert.Nil(t, l)
		assert.Contains(t, err.Error(), "unsupported logger engine: invalid_engine")
	})
}

func TestSlogLogger(t *testing.T) {
	t.Run("With returns a new SlogLogger instance", func(t *testing.T) {
		l, err := logger.NewLogger(logger.EngineSlog, false)
		require.NoError(t, err)

		child := l.With("key", "value")
		require.NotNil(t, child)
		assert.IsType(t, &logger.SlogLogger{}, child)
		assert.NotSame(t, l, child)
	})

	t.Run("logs structured output to stdout", func(t *testing.T) {
		output := captureStdout(t, func() {
			l, err := logger.NewLogger(logger.EngineSlog, false)
			require.NoError(t, err)
			l.Info("test message", "user_id", "123")
		})

		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "user_id=123")
	})

	t.Run("formatted logs work correctly", func(t *testing.T) {
		output := captureStdout(t, func() {
			l, err := logger.NewLogger(logger.EngineSlog, false)
			require.NoError(t, err)
			l.Infof("user %s logged in", "admin")
		})

		assert.Contains(t, output, "user admin logged in")
	})

	t.Run("Sync returns no error", func(t *testing.T) {
		l, err := logger.NewLogger(logger.EngineSlog, false)
		require.NoError(t, err)
		assert.NoError(t, l.Sync())
	})
}

func TestZapLogger(t *testing.T) {
	t.Run("With returns a new ZapLogger instance", func(t *testing.T) {
		l, err := logger.NewLogger(logger.EngineZap, false)
		require.NoError(t, err)

		child := l.With("key", "value")
		require.NotNil(t, child)
		assert.IsType(t, &logger.ZapLogger{}, child)
		assert.NotSame(t, l, child)
	})

	t.Run("logs structured output to stdout", func(t *testing.T) {
		output := captureStdout(t, func() {
			l, err := logger.NewLogger(logger.EngineZap, false)
			require.NoError(t, err)
			l.Info("test message", "user_id", "123")
		})

		assert.Contains(t, output, "test message")
		assert.Contains(t, output, "user_id")
		assert.Contains(t, output, "123")
	})

	t.Run("formatted logs work correctly", func(t *testing.T) {
		output := captureStdout(t, func() {
			l, err := logger.NewLogger(logger.EngineZap, false)
			require.NoError(t, err)
			l.Infof("user %s logged in", "admin")
		})

		assert.Contains(t, output, "user admin logged in")
	})

	t.Run("Sync does not panic on pipe", func(t *testing.T) {
		// Zap's Sync calls fsync, which fails on an os.Pipe with "invalid argument".
		// We test that it doesn't panic, as the error is expected in this specific mock scenario.
		assert.NotPanics(t, func() {
			l, err := logger.NewLogger(logger.EngineZap, false)
			require.NoError(t, err)
			_ = l.Sync()
		})
	})
}

func TestNopLogger(t *testing.T) {
	l := logger.NewNopLogger()

	t.Run("With returns a NopLogger instance", func(t *testing.T) {
		child := l.With("key", "value")
		assert.IsType(t, &logger.NopLogger{}, child)
		// Note: assert.NotSame is intentionally omitted here.
		// In Go, empty structs (zero size) may share the same memory address due to compiler optimizations.
	})

	t.Run("Sugar returns itself", func(t *testing.T) {
		assert.Same(t, l, l.Sugar())
	})

	t.Run("all log levels do not panic", func(t *testing.T) {
		assert.NotPanics(t, func() { l.Debug("debug") })
		assert.NotPanics(t, func() { l.Info("info") })
		assert.NotPanics(t, func() { l.Warn("warn") })
		assert.NotPanics(t, func() { l.Error("error") })
		assert.NotPanics(t, func() { l.Debugf("debug %s", "f") })
		assert.NotPanics(t, func() { l.Infof("info %s", "f") })
		assert.NotPanics(t, func() { l.Warnf("warn %s", "f") })
		assert.NotPanics(t, func() { l.Errorf("error %s", "f") })
	})

	t.Run("Sync returns no error", func(t *testing.T) {
		assert.NoError(t, l.Sync())
	})
}
