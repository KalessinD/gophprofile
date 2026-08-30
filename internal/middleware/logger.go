package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/go-chi/chi/middleware"
)

type (
	responseData struct {
		status int
		size   int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

const LoggerKey ContextKey = "logger"

// Extracts Logger from context
func GetLogger(ctx context.Context) logger.Logger {
	if log, ok := ctx.Value(LoggerKey).(logger.Logger); ok {
		return log
	}
	return nil
}

/*
Добавляет логгер в контекст
*/
func AddLoggerToContext(parentCtx context.Context, log logger.Logger) context.Context {
	return context.WithValue(parentCtx, LoggerKey, log)
}

/*
Обёртка над http.ResponseWriter.Write
*/
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

/*
Обёртка над http.ResponseWriter.Writeheader
*/
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

/*
Мидлварь для добавления логгера с кастомными полями в контекст.

Добавляемые поля:
  - method - HTTP method
  - path - HTTP path
  - remote_addr - remote IP address
  - duration - время выполнения основной части запроса
  - status - HTTP status code
  - response_size - HTTP response size
  - request-content-encoding - HTTP заголовок Content-Encoding из запроса
  - request-accept-encoding - HTTP заголовок Accept-Encoding из запроса
  - responsecontent-encoding - HTTP заголовок Content-Encoding из ответа
  - response-accept-encoding - HTTP заголовок Accept-Encoding из ответа
*/
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

			ctx := AddLoggerToContext(r.Context(), log.With("request_id", middleware.GetReqID(r.Context())))

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
