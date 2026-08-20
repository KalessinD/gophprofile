package middleware

import (
	"context"
	"net/http"

	ch "github.com/KalessinD/gophprofile/internal/common"
)

const UserIDHeader = "X-User-ID"

// GetUserID extracts the user ID from the context.
// If not found, returns an empty string.
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// UserIDMiddleware extracts the X-User-ID header and puts it into the request context.
// If the header is missing or empty, it immediately returns HTTP 400 Bad Request.
func UserIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(UserIDHeader)
		if userID == "" {
			w.Header().Set("Content-Type", ch.AppJSONContentType)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "X-User-ID header is required"}`))
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
