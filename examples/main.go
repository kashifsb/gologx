package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/kashifsb/gologx/pkg/logger"
)

func main() {
	// ──────────────────────────────────────────
	// 1. Initialize the logger
	// ──────────────────────────────────────────
	pp := true
	appLogger := logger.New(logger.Config{
		Level:       logger.TraceLevel, // show everything for demo
		PrettyPrint: &pp,
		WithCaller:  true,
		ServiceName: "golog-demo",
	})

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Golog Logger — Format Showcase                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ──────────────────────────────────────────
	// 2. Basic log levels
	// ──────────────────────────────────────────
	printSection("Basic Log Levels")

	appLogger.Trace().Msg("This is a TRACE message — finest granularity")
	appLogger.Debug().Msg("This is a DEBUG message — diagnostic detail")
	appLogger.Info().Msg("This is an INFO message — normal operation")
	appLogger.Warn().Msg("This is a WARN message — something looks off")
	appLogger.Error().Msg("This is an ERROR message — something failed")

	// ──────────────────────────────────────────
	// 3. Structured fields
	// ──────────────────────────────────────────
	printSection("Structured Fields")

	appLogger.Info().
		Str("user", "kashifsb").
		Str("action", "login").
		Str("ip", "192.168.1.42").
		Int("attempt", 1).
		Bool("mfa", true).
		Msg("User authentication successful")

	appLogger.Debug().
		Str("query", "SELECT * FROM users WHERE active = true").
		Int("params", 0).
		Float64("cost_estimate", 0.0034).
		Msg("Query plan analyzed")

	// ──────────────────────────────────────────
	// 4. Error logging with context
	// ──────────────────────────────────────────
	printSection("Error Logging")

	dbErr := errors.New("connection refused: dial tcp 10.0.0.5:5432")
	appLogger.Error().
		Err(dbErr).
		Str("host", "10.0.0.5").
		Int("port", 5432).
		Str("database", "myapp_prod").
		Msg("Failed to connect to database")

	wrappedErr := fmt.Errorf("user service: %w", fmt.Errorf("repository: %w", dbErr))
	appLogger.Error().
		Err(wrappedErr).
		Str("operation", "GetUserByID").
		Str("user_id", "usr_abc123").
		Msg("Service operation failed")

	// ──────────────────────────────────────────
	// 5. Sub-loggers with context
	// ──────────────────────────────────────────
	printSection("Sub-loggers (Component / Request Scoped)")

	dbLogger := logger.WithComponent(appLogger, "database")
	dbLogger.Info().Str("driver", "pgx").Str("host", "localhost:5432").Msg("Connection pool initialized")
	dbLogger.Debug().Int("pool_size", 25).Int("idle", 10).Msg("Pool stats")

	authLogger := logger.WithComponent(appLogger, "auth")
	authLogger.Info().Str("provider", "kerberos").Msg("Auth provider configured")

	reqLogger := logger.WithRequestID(appLogger, "req-7f3a-4b2c-9d1e")
	reqLogger.Info().Str("method", "POST").Str("path", "/api/v1/entries").Msg("Processing request")
	reqLogger.Debug().Str("content_type", "application/json").Int("body_size", 1024).Msg("Request body parsed")

	traceLogger := logger.WithTraceID(appLogger, "trace-abc123def456")
	traceLogger.Info().Msg("Distributed trace started")

	// ──────────────────────────────────────────
	// 6. Context-based logging
	// ──────────────────────────────────────────
	printSection("Context-based Logging")

	ctx := logger.WithContext(context.Background(), reqLogger)
	simulateHandler(ctx)

	// ──────────────────────────────────────────
	// 7. Helper functions
	// ──────────────────────────────────────────
	printSection("Helper Functions")

	logger.LogSuccess(appLogger, "Database migration completed")
	logger.LogSuccessf(appLogger, "Deployed version %s to %s", "v2.4.1", "production")

	logger.LogRequest(appLogger, "GET", "/api/v1/users/42", "kashifsb")
	logger.LogRequest(appLogger, "POST", "/api/v1/entries", "servicebot")

	logger.LogResponse(appLogger, "GET", "/api/v1/users/42", 200, 12*time.Millisecond, nil)
	logger.LogResponse(appLogger, "POST", "/api/v1/entries", 201, 45*time.Millisecond, nil)
	logger.LogResponse(appLogger, "GET", "/api/v1/secrets", 403, 2*time.Millisecond, errors.New("insufficient permissions"))
	logger.LogResponse(appLogger, "GET", "/api/v1/missing", 404, 1*time.Millisecond, errors.New("resource not found"))
	logger.LogResponse(appLogger, "POST", "/api/v1/entries", 500, 3*time.Second, errors.New("deadlock detected"))

	logger.LogDBQuery(appLogger, "SELECT", "users", 3*time.Millisecond, 42)
	logger.LogDBQuery(appLogger, "INSERT", "time_entries", 15*time.Millisecond, 1)
	logger.LogDBQuery(appLogger, "DELETE", "old_sessions", 200*time.Millisecond, 1337)

	logger.LogServiceError(appLogger, "UserService", "CreateUser", errors.New("duplicate email"), map[string]interface{}{
		"email":    "user@example.com",
		"provider": "internal",
	})

	logger.LogServiceDebug(appLogger, "CacheService", "Get", "Cache lookup completed", map[string]interface{}{
		"key":    "user:42:profile",
		"hit":    true,
		"ttl_ms": 45000,
	})

	// ──────────────────────────────────────────
	// 8. Redaction
	// ──────────────────────────────────────────
	printSection("Sensitive Field Redaction")

	event := appLogger.Info()
	event = logger.RedactedStr(event, "username", "kashifsb")
	event = logger.RedactedStr(event, "password", "super_secret_pass_123")
	event = logger.RedactedStr(event, "api_key", "sk-proj-abc123def456ghi789")
	event = logger.RedactedStr(event, "email", "user@example.com")
	event.Msg("Login attempt with redacted credentials")

	formData := map[string]string{
		"username":     "kashifsb",
		"password":     "super_secret_pass_123",
		"access_token": "eyJhbGciOiJIUzI1NiJ9",
		"action":       "login",
	}
	redacted := logger.RedactMap(formData)
	appLogger.Info().Interface("form_data", redacted).Msg("Form submission (redacted)")

	headers := map[string][]string{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer eyJhbGciOiJSUzI1NiJ9.longtoken"},
		"X-Request-ID":  {"req-7f3a-4b2c"},
		"X-Api-Key":     {"sk-live-abcdef123456"},
	}
	redactedHeaders := logger.RedactHeaders(headers)
	appLogger.Info().Interface("headers", redactedHeaders).Msg("Request headers (redacted)")

	// ──────────────────────────────────────────
	// 9. HTTP Middleware
	// ──────────────────────────────────────────
	printSection("HTTP Middleware (simulated requests)")

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		reqLog := logger.FromContext(r.Context())
		reqLog.Debug().Msg("Health check — all systems operational")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		reqLog := logger.FromContext(r.Context())
		reqLog.Info().Int("count", 42).Msg("Fetched user list")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users":[],"total":42}`))
	})

	mux.HandleFunc("/api/v1/error", func(w http.ResponseWriter, r *http.Request) {
		reqLog := logger.FromContext(r.Context())
		reqLog.Error().Err(errors.New("simulated failure")).Msg("Something went wrong")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	})

	handler := logger.HTTPMiddleware(appLogger)(mux)

	simulateRequest(handler, "GET", "/api/v1/health", "")
	fmt.Println()
	simulateRequest(handler, "GET", "/api/v1/users", "req-user-list-001")
	fmt.Println()
	simulateRequest(handler, "GET", "/api/v1/error", "req-error-sim-002")

	// ──────────────────────────────────────────
	// 10. JSON output mode
	// ──────────────────────────────────────────
	printSection("JSON Output Mode (for production / log aggregators)")

	ppFalse := false
	jsonLogger := logger.New(logger.Config{
		Level:       logger.InfoLevel,
		PrettyPrint: &ppFalse,
		WithCaller:  true,
		ServiceName: "golog-demo",
	})

	jsonLogger.Info().
		Str("environment", "production").
		Str("version", "2.4.1").
		Msg("Application started")

	jsonLogger.Error().
		Err(errors.New("connection timeout")).
		Str("service", "redis").
		Str("host", "redis.internal:6379").
		Msg("Cache unavailable")

	// ──────────────────────────────────────────
	// Done!
	// ──────────────────────────────────────────
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    All formats demonstrated!                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
}

func simulateHandler(ctx context.Context) {
	l := logger.FromContext(ctx)
	l.Info().Msg("Handler retrieved logger from context")
	l.Debug().Str("step", "validation").Msg("Validating request payload")
	l.Info().Str("step", "processing").Dur("elapsed", 42*time.Millisecond).Msg("Processing complete")
}

func simulateRequest(handler http.Handler, method, path, requestID string) {
	req := httptest.NewRequest(method, path, nil)
	if requestID != "" {
		req.Header.Set("X-Request-ID", requestID)
	}
	req.RemoteAddr = "192.168.1.100:54321"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func printSection(title string) {
	fmt.Println()
	pad := 58 - len(title)
	if pad < 0 {
		pad = 0
	}
	line := ""
	for i := 0; i < pad; i++ {
		line += "─"
	}
	fmt.Printf("── %s %s\n\n", title, line)
}
