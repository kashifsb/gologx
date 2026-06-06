# golog

![Go CI](https://github.com/kashifsb/gologx/actions/workflows/ci.yml/badge.svg)

Universal structured logging library for Go — stdlib-only, zero dependencies.

## Features

- **Leveled logging** — Trace, Debug, Info, Warn, Error, Fatal
- **Pretty + JSON output** — colored human-readable or structured JSON
- **Auto-detection** — TTY gets pretty mode, non-TTY gets JSON
- **Environment configuration** — LOG_LEVEL, LOG_FORMAT, LOG_CALLER, NO_COLOR, TERM
- **Sub-loggers** — persistent fields via WithComponent, WithRequestID, WithTraceID
- **Field redaction** — auto-detect and mask sensitive keys (passwords, tokens, API keys)
- **HTTP middleware** — request/response logging with auto-generated request IDs
- **Context integration** — store/retrieve loggers from `context.Context`
- **Convenience helpers** — LogSuccess, LogSuccessf, LogRequest, LogResponse, LogDBQuery, LogServiceError, LogServiceDebug
- **Thread-safe** — safe for concurrent use
- **Fluent API** — chainable Event methods for structured fields

## Quick Start

```go
package main

import "github.com/kashifsb/gologx/pkg/logger"

func main() {
    logger.InitGlobalFromEnv()
    log := logger.New(logger.Config{})

    log.Info().
        Str("user", "alice").
        Int("attempt", 1).
        Msg("User authenticated")
}
```

## Configuration

| Environment Variable | Values | Default |
|---------------------|--------|---------|
| `LOG_LEVEL` | trace, debug, info, warn, error, fatal | info |
| `LOG_FORMAT` | pretty, json | auto (TTY=pretty, else JSON) |
| `LOG_CALLER` | true, false | true |
| `NO_COLOR` | (any value) | unset |
| `TERM` | dumb (disables color) | unset |

Programmatic configuration via `Config`:

```go
pp := true
log := logger.New(logger.Config{
    Level:       logger.DebugLevel,
    PrettyPrint: &pp,           // nil = auto-detect
    WithCaller:  true,
    NoColor:     false,
    ServiceName: "my-service",  // optional service identifier
})
```

## Output Formats

### Pretty Mode (TTY)

```
2026/03/29 14:30:42 UTC INF User authenticated user=alice attempt=1 (main.go:12)
2026/03/29 14:30:42 UTC ERR Connection failed error=timeout host=db.internal (main.go:15)
```

### JSON Mode (production)

```json
{"level":"info","time":"2026-03-29T14:30:42Z","message":"User authenticated","user":"alice","attempt":1,"caller":"main.go:12"}
{"level":"error","time":"2026-03-29T14:30:42Z","message":"Connection failed","error":"timeout","host":"db.internal","caller":"main.go:15"}
```

## Fluent Event API

Log methods return an `*Event` that supports chaining:

```go
log.Info().
    Str("user", "baleeghu").
    Int("attempt", 1).
    Bool("mfa", true).
    Float64("cost", 0.0034).
    Dur("elapsed", 42*time.Millisecond).
    Err(err).
    Msg("User authentication successful")

// Or with formatted messages:
log.Info().Str("user", "alice").Msgf("Deployed version %s", "v2.4.1")
```

## Sub-loggers

Create child loggers with persistent fields that appear on every log entry:

```go
dbLog := logger.WithComponent(log, "database")
dbLog.Info().Str("driver", "pgx").Str("host", "localhost:5432").Msg("Connection pool initialized")
dbLog.Debug().Int("pool_size", 25).Int("idle", 10).Msg("Pool stats")

reqLog := logger.WithRequestID(log, "req-7f3a-4b2c-9d1e")
reqLog.Info().Str("method", "POST").Str("path", "/api/v1/entries").Msg("Processing request")

traceLog := logger.WithTraceID(log, "trace-abc123def456")
traceLog.Info().Msg("Distributed trace started")

// Or use the fluent builder for arbitrary fields:
custom := log.With().
    Str("service", "auth").
    Str("version", "2.1").
    Logger()
```

## Context Integration

Store and retrieve loggers from `context.Context`:

```go
// Store a logger in context
ctx := logger.WithContext(context.Background(), reqLog)

// Retrieve it in any downstream function
func handler(ctx context.Context) {
    l := logger.FromContext(ctx)
    l.Info().Msg("Handler retrieved logger from context")
    l.Debug().Str("step", "validation").Msg("Validating request payload")
    l.Info().Str("step", "processing").Dur("elapsed", 42*time.Millisecond).Msg("Processing complete")
}
```

`FromContext` falls back to the default global logger if none is stored in the context.

## Convenience Helpers

```go
logger.LogSuccess(log, "Database migration completed")
logger.LogSuccessf(log, "Deployed version %s to %s", "v2.4.1", "production")

logger.LogRequest(log, "GET", "/api/v1/users/42", "baleeghu")
logger.LogRequest(log, "POST", "/api/v1/entries", "servicebot")

logger.LogResponse(log, "GET", "/api/v1/users/42", 200, 12*time.Millisecond, nil)
logger.LogResponse(log, "POST", "/api/v1/entries", 500, 3*time.Second, errors.New("deadlock detected"))

logger.LogDBQuery(log, "SELECT", "users", 3*time.Millisecond, 42)
logger.LogDBQuery(log, "INSERT", "time_entries", 15*time.Millisecond, 1)

logger.LogServiceError(log, "UserService", "CreateUser", errors.New("duplicate email"), map[string]interface{}{
    "email":    "user@example.com",
    "provider": "internal",
})

logger.LogServiceDebug(log, "CacheService", "Get", "Cache lookup completed", map[string]interface{}{
    "key":    "user:42:profile",
    "hit":    true,
    "ttl_ms": 45000,
})
```

`LogResponse` sets the level based on status code: 5xx = Error, 4xx = Warn, else Info.

## Field Redaction

Sensitive keys are automatically detected (case-insensitive, substring match):

`password`, `secret`, `token`, `access_token`, `refresh_token`, `authorization`, `api_key`, `apikey`, `api-key`, `credit_card`, `ssn`

| Input | RedactValue Output |
|-------|-------------------|
| `""` | `****` |
| `"ab"` | `**` |
| `"secret"` | `s*****` |
| `"mysecrettoken"` | `m************n` |

```go
// Single field — redacts automatically if key is sensitive
event := log.Info()
event = logger.RedactedStr(event, "username", "baleeghu")
event = logger.RedactedStr(event, "password", "super_secret_pass_123")
event = logger.RedactedStr(event, "api_key", "sk-proj-abc123def456ghi789")
event.Msg("Login attempt with redacted credentials")

// Multiple fields from a map
logger.RedactedFields(log.Info(), map[string]string{
    "username":     "baleeghu",
    "password":     "hunter2",
    "access_token": "eyJhbGciOiJIUzI1NiJ9",
}).Msg("Form submission")

// Standalone map redaction
safe := logger.RedactMap(formData)

// HTTP header redaction
safe := logger.RedactHeaders(req.Header)
```

## HTTP Middleware

```go
mux := http.NewServeMux()

mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
    reqLog := logger.FromContext(r.Context())
    reqLog.Debug().Msg("Health check — all systems operational")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"healthy"}`))
})

handler := logger.HTTPMiddleware(log)(mux)
http.ListenAndServe(":8080", handler)
```

The middleware:
1. Extracts or generates `X-Request-ID` header (UUID v4 if missing)
2. Creates a sub-logger with request_id, method, path, remote_addr
3. Logs "Request started" at Info level
4. Stores the logger in context (retrieve via `logger.FromContext(r.Context())`)
5. Sets `X-Request-ID` on response headers
6. Logs response with status, bytes written, and duration

## Development

```bash
make test      # go test -v -race -count=1 ./...
make vet       # go vet ./...
make build     # go build ./...
make example   # go run ./examples/
make check     # vet + test
make clean     # clean build/test cache
```

## Project Structure

```
golog/
├── go.mod
├── Makefile
├── golog.go                     # Package documentation
├── examples/
│   └── main.go                  # Comprehensive usage demo
└── pkg/logger/
    ├── logger.go                # Logger, Config, constructors
    ├── event.go                 # Fluent Event API (Str, Int, Bool, Dur, Msg, etc.)
    ├── level.go                 # Level type and parsing
    ├── context_builder.go       # Sub-logger builder (With().Str().Logger())
    ├── helpers.go               # WithComponent, WithContext, LogSuccess, LogRequest, etc.
    ├── redact.go                # Sensitive key detection and masking
    ├── middleware.go            # HTTP middleware
    ├── pretty.go                # Colored pretty-print formatter
    ├── uuid.go                  # UUID v4 generation (crypto/rand)
    └── logger_test.go           # Tests
```
