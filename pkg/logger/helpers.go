package logger

import (
	"context"
	"time"
)

// contextKey is an unexported type used for storing the logger in context.
type contextKey struct{}

// WithContext stores a Logger in the given context.
func WithContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext retrieves a Logger from the context.
// Falls back to the default package logger if none is found.
func FromContext(ctx context.Context) Logger {
	if l, ok := ctx.Value(contextKey{}).(Logger); ok {
		return l
	}
	return defaultLogger
}

// WithRequestID returns a sub-logger with the given request ID attached.
func WithRequestID(logger Logger, requestID string) Logger {
	return logger.With().Str("request_id", requestID).Logger()
}

// WithTraceID returns a sub-logger with a distributed trace ID attached.
func WithTraceID(logger Logger, traceID string) Logger {
	return logger.With().Str("trace_id", traceID).Logger()
}

// WithComponent returns a sub-logger scoped to a specific component name.
func WithComponent(logger Logger, component string) Logger {
	return logger.With().Str("component", component).Logger()
}

// ──────────────────────────────────────────────
// Convenience logging functions
// ──────────────────────────────────────────────

// LogSuccess logs a success message at Info level.
// CallerSkipFrame(1) ensures the caller points to the actual call site,
// not this helper function.
func LogSuccess(logger Logger, message string) {
	logger.Info().
		CallerSkipFrame(1).
		Str("result", "success").
		Msg(message)
}

// LogSuccessf logs a formatted success message at Info level.
func LogSuccessf(logger Logger, format string, args ...interface{}) {
	logger.Info().
		CallerSkipFrame(1).
		Str("result", "success").
		Msgf(format, args...)
}

// LogRequest logs an incoming HTTP or gRPC request.
func LogRequest(logger Logger, method, path, user string) {
	logger.Info().
		CallerSkipFrame(1).
		Str("method", method).
		Str("path", path).
		Str("user", user).
		Msg("Request received")
}

// LogResponse logs an outgoing response, choosing the level based on status code.
func LogResponse(logger Logger, method, path string, statusCode int, duration time.Duration, err error) {
	var event *Event

	switch {
	case statusCode >= 500:
		event = logger.Error()
	case statusCode >= 400:
		event = logger.Warn()
	default:
		event = logger.Info()
	}

	event = event.CallerSkipFrame(1)

	if err != nil {
		event = event.Err(err)
	}

	event.
		Str("method", method).
		Str("path", path).
		Int("status", statusCode).
		Dur("duration", duration).
		Msg("Response sent")
}

// LogDBQuery logs a database query at Debug level.
func LogDBQuery(logger Logger, operation, table string, duration time.Duration, rowsAffected int64) {
	logger.Debug().
		CallerSkipFrame(1).
		Str("operation", operation).
		Str("table", table).
		Dur("duration", duration).
		Int64("rows_affected", rowsAffected).
		Msg("Database query executed")
}

// LogServiceError logs a service-level error with structured context fields.
func LogServiceError(logger Logger, service, operation string, err error, fields map[string]interface{}) {
	event := logger.Error().
		CallerSkipFrame(1).
		Err(err).
		Str("service", service).
		Str("operation", operation)

	for k, v := range fields {
		event = event.Interface(k, v)
	}

	event.Msg("Service operation failed")
}

// LogServiceDebug logs service-level debug information with structured context fields.
func LogServiceDebug(logger Logger, service, operation, message string, fields map[string]interface{}) {
	event := logger.Debug().
		CallerSkipFrame(1).
		Str("service", service).
		Str("operation", operation)

	for k, v := range fields {
		event = event.Interface(k, v)
	}

	event.Msg(message)
}
