package logger

import (
	"net/http"
	"time"
)

// ──────────────────────────────────────────────
// HTTP Middleware
// ──────────────────────────────────────────────

// HTTPMiddleware returns an http.Handler middleware that:
//   - Extracts or generates a request ID (X-Request-ID header).
//   - Creates a sub-logger with request metadata and stores it in context.
//   - Logs the request on entry and the response on completion.
func HTTPMiddleware(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Resolve request ID.
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = newUUID()
			}

			// Build a request-scoped logger.
			reqLogger := logger.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Logger()

			reqLogger.Info().Msg("Request started")

			// Store logger in context so downstream handlers can use FromContext().
			ctx := WithContext(r.Context(), reqLogger)
			r = r.WithContext(ctx)

			// Set the request ID on the response for traceability.
			w.Header().Set("X-Request-ID", requestID)

			// Wrap the ResponseWriter to capture the status code.
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Choose log level based on status code.
			var event *Event
			switch {
			case rw.statusCode >= 500:
				event = reqLogger.Error()
			case rw.statusCode >= 400:
				event = reqLogger.Warn()
			default:
				event = reqLogger.Info()
			}

			event.
				Int("status", rw.statusCode).
				Int("bytes", rw.bytesWritten).
				Dur("duration", duration).
				Msg("Request completed")
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Flush supports streaming responses if the underlying writer implements http.Flusher.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
