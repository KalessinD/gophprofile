package config

import (
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Returns true if YP_ENV environment is set to "prod", which means the production environment.
//
// Otherwise returns false.
func IsProduction() bool {
	return os.Getenv("YP_ENV") == "prod"
}

// Returns true if YP_ENV environment is not set to "prod", which means the production environment.
//
// Otherwise returns false.
func IsDevelopment() bool {
	return !IsProduction()
}

// Returns OS Environment
func GetEnvOrFallback[T any](key string, fallback T) T {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}

	// Casting string to T.
	var result any
	var err error

	switch any(fallback).(type) {
	case string:
		result = valStr
	case int:
		result, err = strconv.Atoi(valStr)
	case float64:
		result, err = strconv.ParseFloat(valStr, 64)
	case bool:
		result, err = strconv.ParseBool(valStr)
	case time.Duration:
		result, err = time.ParseDuration(valStr)
	case Duration:
		var d Duration
		err = d.Set(valStr) // parse string ("2" or "2s")
		if err == nil {
			result = d
		}
	case []byte:
		result = []byte(valStr)
	case io.Reader:
		result = strings.NewReader(valStr)
	case context.Context:
		result = context.Background()
	default: // not supported type
		return fallback
	}

	if err != nil {
		return fallback
	}

	if val, ok := result.(T); ok {
		return val
	}

	return fallback
}
