package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// https://github.com/gin-contrib/requestid
func RequestID(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			w.Header().Set("X-Request-ID", requestID)

			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

			loggerWithID := logger.With().Str("request_id", requestID).Logger()
			ctx = loggerWithID.WithContext(ctx)

			loggerWithID.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Msg("request started")

			next.ServeHTTP(w, r.WithContext(ctx))

			duration := time.Since(start)
			loggerWithID.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int64("duration_ms", duration.Milliseconds()).
				Dur("duration", duration).
				Msg("request completed")
		})
	}
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
