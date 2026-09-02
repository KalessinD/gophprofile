package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/go-chi/chi/middleware"
	"go.opentelemetry.io/otel/trace"
)

type (
	// responseData holds the status code and response size for logging purposes.
	responseData struct {
		status int
		size   int
	}

	// loggingResponseWriter is a custom http.ResponseWriter that wraps the original
	// response writer to intercept and record response data.
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

const LoggerKey ContextKey = "logger"

// GetLogger extracts the Logger instance from the provided context.
// If no logger is found, it returns nil.
func GetLogger(ctx context.Context) logger.Logger {
	if log, ok := ctx.Value(LoggerKey).(logger.Logger); ok {
		return log
	}
	return nil
}

// AddLoggerToContext adds a Logger instance to the parent context.
func AddLoggerToContext(parentCtx context.Context, log logger.Logger) context.Context {
	return context.WithValue(parentCtx, LoggerKey, log)
}

// Write implements io.Writer. It writes to the underlying ResponseWriter
// and accumulates the total number of bytes written.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

// WriteHeader implements http.ResponseWriter. It delegates the status code
// to the underlying ResponseWriter and records it for logging.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// Logger returns a middleware that injects a structured logger with contextual
// fields into the request context and logs the request completion.
//
// Injected fields (available to downstream handlers via GetLogger):
//   - request_id: Unique identifier for the request.
//   - trace_id: OpenTelemetry Trace ID for log correlation with Jaeger (if valid).
//   - span_id: OpenTelemetry Span ID for log correlation with Jaeger (if valid).
//
// Logged fields upon request completion:
//   - method: HTTP method.
//   - path: HTTP path.
//   - remote_addr: Remote IP address.
//   - duration: Time taken to process the request.
//   - status: HTTP response status code.
//   - response_size: Size of the HTTP response in bytes.
//   - request-content-encoding: Content-Encoding header from the request.
//   - request-accept-encoding: Accept-Encoding header from the request.
//   - response-Content-Encoding: Content-Encoding header of the response.
//   - response-Accept-Encoding: Accept-Encoding header of the response.
func Logger(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			responseData := &responseData{
				status: 0,
				size:   0,
			}

			lw := &loggingResponseWriter{
				ResponseWriter: w,
				responseData:   responseData,
			}

			// Extract OpenTelemetry trace context for log correlation
			spanContext := trace.SpanContextFromContext(r.Context())
			logWithFields := log.With("request_id", middleware.GetReqID(r.Context()))

			// Only add trace fields if a valid trace context exists
			if spanContext.IsValid() {
				logWithFields = logWithFields.With(
					"trace_id", spanContext.TraceID().String(),
					"span_id", spanContext.SpanID().String(),
				)
			}

			ctx := AddLoggerToContext(r.Context(), logWithFields)

			next.ServeHTTP(lw, r.WithContext(ctx))

			GetLogger(ctx).Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"duration", time.Since(start),
				"status", responseData.status,
				"response_size", responseData.size,
				"request-content-encoding", r.Header.Values("Content-Encoding"),
				"request-accept-encoding", r.Header.Values("Accept-Encoding"),
				"response-Content-Encoding", w.Header().Values("Content-Encoding"),
				"response-Accept-Encoding", w.Header().Values("Accept-Encoding"),
			)
		})
	}
}
