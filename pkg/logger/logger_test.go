package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ──────────────────────────────────────────────
// Logger creation tests
// ──────────────────────────────────────────────

func TestNew_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		WithCaller:  false,
	})

	logger.Info().Str("foo", "bar").Msg("hello world")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON output, got: %s", buf.String())
	}

	if entry["message"] != "hello world" {
		t.Errorf("expected message 'hello world', got %v", entry["message"])
	}
	if entry["foo"] != "bar" {
		t.Errorf("expected foo='bar', got %v", entry["foo"])
	}
}

func TestNew_WithServiceName(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		ServiceName: "test-service",
	})

	logger.Info().Msg("test")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if entry["service"] != "test-service" {
		t.Errorf("expected service='test-service', got %v", entry["service"])
	}
}

func TestNew_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       WarnLevel,
		PrettyPrint: &pp,
	})

	logger.Info().Msg("should not appear")
	if buf.Len() > 0 {
		t.Error("info message should be filtered at warn level")
	}

	logger.Warn().Msg("should appear")
	if buf.Len() == 0 {
		t.Error("warn message should not be filtered at warn level")
	}
}

// ──────────────────────────────────────────────
// Pretty writer tests
// ──────────────────────────────────────────────

func TestPrettyWriter_Output(t *testing.T) {
	var buf bytes.Buffer
	pp := true

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		NoColor:     true,
		WithCaller:  false,
	})

	logger.Info().Msg("pretty test")

	output := buf.String()
	if !strings.Contains(output, "INF") {
		t.Errorf("expected level badge in output, got: %s", output)
	}
	if !strings.Contains(output, "pretty test") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "UTC") {
		t.Errorf("expected UTC timestamp in output, got: %s", output)
	}
}

func TestPrettyWriter_ErrorHighlighting(t *testing.T) {
	var buf bytes.Buffer
	pp := true

	logger := New(Config{
		Output:      &buf,
		Level:       ErrorLevel,
		PrettyPrint: &pp,
		NoColor:     true,
		WithCaller:  false,
	})

	logger.Error().Err(errForTest("something broke")).Msg("failure")

	output := buf.String()
	if !strings.Contains(output, "ERR") {
		t.Errorf("expected ERROR badge, got: %s", output)
	}
	if !strings.Contains(output, "something broke") {
		t.Errorf("expected error text, got: %s", output)
	}
}

func TestFormatLevel_AllLevels(t *testing.T) {
	w := &prettyWriter{noColor: true}

	tests := []struct {
		level    string
		contains string
	}{
		{"trace", "TRC"},
		{"debug", "DBG"},
		{"info", "INF"},
		{"warn", "WAR"},
		{"error", "ERR"},
		{"fatal", "FTL"},
		{"panic", "PNC"},
		{"custom", "CUS"},
	}

	for _, tt := range tests {
		result := w.formatLevel(tt.level)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("formatLevel(%q) = %q, expected to contain %q", tt.level, result, tt.contains)
		}
	}
}

func TestFormatCaller(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/project/pkg/handler/user.go:42", "handler/user.go:42"},
		{"main.go:10", "main.go:10"},
		{"a/b.go:1", "a/b.go:1"},
	}

	for _, tt := range tests {
		result := formatCaller(tt.input)
		if result != tt.expected {
			t.Errorf("formatCaller(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestPrettyWriter_NoANSI_WhenNoColor(t *testing.T) {
	var buf bytes.Buffer
	pp := true

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		NoColor:     true,
	})

	logger.Info().Msg("no color test")
	if strings.Contains(buf.String(), "\033[") {
		t.Error("NoColor output should not contain ANSI escape codes")
	}
}

func TestPrettyWriter_ANSI_WhenColorEnabled(t *testing.T) {
	var buf bytes.Buffer
	pp := true

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		NoColor:     false,
	})

	logger.Info().Msg("color test")
	if !strings.Contains(buf.String(), "\033[") {
		t.Error("colored output should contain ANSI escape codes")
	}
}

func TestPrettyWriter_MalformedInput(t *testing.T) {
	var buf bytes.Buffer
	w := newPrettyWriter(&buf, true)

	_, err := w.Write([]byte("not json at all"))
	if err != nil {
		t.Fatalf("malformed input should not error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("malformed input should still produce output")
	}
}

// ──────────────────────────────────────────────
// Context helpers tests
// ──────────────────────────────────────────────

func TestWithContext_and_FromContext(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
	})

	ctx := WithContext(context.Background(), logger)
	retrieved := FromContext(ctx)

	retrieved.Info().Msg("from context")

	if buf.Len() == 0 {
		t.Error("expected log output from context-retrieved logger")
	}

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}
	if entry["message"] != "from context" {
		t.Errorf("unexpected message: %v", entry["message"])
	}
}

func TestFromContext_FallsBackToGlobal(t *testing.T) {
	logger := FromContext(context.Background())
	logger.Info().Msg("fallback test")
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	base := New(Config{Output: &buf, Level: InfoLevel, PrettyPrint: &pp})
	sub := WithComponent(base, "auth")
	sub.Info().Msg("test")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["component"] != "auth" {
		t.Errorf("expected component='auth', got %v", entry["component"])
	}
}

func TestWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	base := New(Config{Output: &buf, Level: InfoLevel, PrettyPrint: &pp})
	sub := WithRequestID(base, "req-123")
	sub.Info().Msg("test")

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["request_id"] != "req-123" {
		t.Errorf("expected request_id='req-123', got %v", entry["request_id"])
	}
}

func TestSubLoggerPersistence(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	base := New(Config{Output: &buf, Level: InfoLevel, PrettyPrint: &pp})
	sub := WithComponent(base, "db")
	sub = WithRequestID(sub, "req-1")

	sub.Info().Msg("first")
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var d1 map[string]interface{}
	json.Unmarshal([]byte(lines[0]), &d1)
	if d1["component"] != "db" || d1["request_id"] != "req-1" {
		t.Error("sub-logger should carry persistent fields")
	}

	sub.Warn().Msg("second")
	lines = strings.Split(strings.TrimSpace(buf.String()), "\n")
	var d2 map[string]interface{}
	json.Unmarshal([]byte(lines[1]), &d2)
	if d2["component"] != "db" || d2["request_id"] != "req-1" {
		t.Error("sub-logger fields should persist across log calls")
	}
}

// ──────────────────────────────────────────────
// Redaction tests
// ──────────────────────────────────────────────

func TestRedactValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"a", "*"},
		{"ab", "**"},
		{"abc", "a**"},
		{"abcdef", "a*****"},
		{"mysecrettoken", "m***********n"},
	}

	for _, tt := range tests {
		result := RedactValue(tt.input)
		if result != tt.expected {
			t.Errorf("RedactValue(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsSensitiveKey(t *testing.T) {
	sensitive := []string{
		"password", "Password", "PASSWORD",
		"api_key", "API_KEY",
		"authorization", "Authorization",
		"db_password", "user_token",
		"x-api-key",
	}
	for _, key := range sensitive {
		if !IsSensitiveKey(key) {
			t.Errorf("expected %q to be sensitive", key)
		}
	}

	notSensitive := []string{"username", "email", "path", "method", "status"}
	for _, key := range notSensitive {
		if IsSensitiveKey(key) {
			t.Errorf("expected %q to NOT be sensitive", key)
		}
	}
}

func TestRedactMap(t *testing.T) {
	input := map[string]string{
		"username": "alice",
		"password": "hunter2",
		"token":    "abc123xyz",
	}

	result := RedactMap(input)

	if result["username"] != "alice" {
		t.Errorf("username should not be redacted, got %q", result["username"])
	}
	if result["password"] == "hunter2" {
		t.Error("password should be redacted")
	}
	if result["token"] == "abc123xyz" {
		t.Error("token should be redacted")
	}
}

func TestRedactedStr_InLogOutput(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		WithCaller:  false,
	})

	event := logger.Info()
	event = RedactedStr(event, "api_key", "sk-12345678")
	event = RedactedStr(event, "user", "alice")
	event.Msg("test redaction")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if entry["api_key"] == "sk-12345678" {
		t.Error("api_key should be redacted in output")
	}
	if entry["user"] != "alice" {
		t.Errorf("user should not be redacted, got %v", entry["user"])
	}
}

// ──────────────────────────────────────────────
// HTTP Middleware tests
// ──────────────────────────────────────────────

func TestHTTPMiddleware(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		WithCaller:  false,
	})

	handler := HTTPMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqLogger := FromContext(r.Context())
		reqLogger.Info().Msg("inside handler")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("X-Request-ID", "test-req-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "test-req-123" {
		t.Errorf("expected X-Request-ID header, got %q", rec.Header().Get("X-Request-ID"))
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 log lines, got %d:\n%s", len(lines), buf.String())
	}

	for i, line := range lines {
		if !strings.Contains(line, "test-req-123") {
			t.Errorf("line %d missing request_id: %s", i, line)
		}
	}
}

func TestHTTPMiddleware_StatusCodeLevels(t *testing.T) {
	tests := []struct {
		statusCode    int
		expectedLevel string
	}{
		{200, "info"},
		{201, "info"},
		{400, "warn"},
		{404, "warn"},
		{500, "error"},
		{503, "error"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		pp := false

		logger := New(Config{
			Output:      &buf,
			Level:       DebugLevel,
			PrettyPrint: &pp,
			WithCaller:  false,
		})

		handler := HTTPMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.statusCode)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, "Request completed") {
				var entry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					t.Fatalf("JSON parse error for status %d: %v", tt.statusCode, err)
				}
				if entry["level"] != tt.expectedLevel {
					t.Errorf("status %d: expected level %q, got %v", tt.statusCode, tt.expectedLevel, entry["level"])
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("status %d: 'Request completed' log line not found", tt.statusCode)
		}
	}
}

func TestHTTPMiddleware_GeneratesRequestID(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
	})

	handler := HTTPMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("expected middleware to generate an X-Request-ID")
	}
}

// ──────────────────────────────────────────────
// Helper function tests
// ──────────────────────────────────────────────

func TestLogSuccess(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		WithCaller:  false,
	})

	LogSuccess(logger, "deployment complete")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if entry["result"] != "success" {
		t.Errorf("expected result='success', got %v", entry["result"])
	}
	msg, _ := entry["message"].(string)
	if !strings.Contains(msg, "deployment complete") {
		t.Errorf("expected message to contain 'deployment complete', got %q", msg)
	}
}

func TestLogResponse_WithError(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		WithCaller:  false,
	})

	LogResponse(logger, "GET", "/api/users", 500, 150*time.Millisecond, errForTest("db timeout"))

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if entry["level"] != "error" {
		t.Errorf("expected level 'error' for 500, got %v", entry["level"])
	}
	if entry["error"] != "db timeout" {
		t.Errorf("expected error 'db timeout', got %v", entry["error"])
	}
}

func TestLogDBQuery(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       DebugLevel,
		PrettyPrint: &pp,
	})

	LogDBQuery(logger, "SELECT", "users", 3*time.Millisecond, 42)

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)

	if entry["operation"] != "SELECT" {
		t.Errorf("expected operation='SELECT', got %v", entry["operation"])
	}
	if entry["table"] != "users" {
		t.Errorf("expected table='users', got %v", entry["table"])
	}
}

func TestLogServiceError(t *testing.T) {
	var buf bytes.Buffer
	pp := false

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
	})

	LogServiceError(logger, "UserService", "CreateUser", errForTest("duplicate email"), map[string]interface{}{
		"email": "user@example.com",
	})

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)

	if entry["service"] != "UserService" {
		t.Errorf("expected service='UserService', got %v", entry["service"])
	}
	if entry["operation"] != "CreateUser" {
		t.Errorf("expected operation='CreateUser', got %v", entry["operation"])
	}
	if entry["level"] != "error" {
		t.Errorf("expected level='error', got %v", entry["level"])
	}
}

// ──────────────────────────────────────────────
// Concurrent safety test
// ──────────────────────────────────────────────

func TestPrettyWriter_ConcurrentWrites(t *testing.T) {
	var buf bytes.Buffer
	pp := true

	logger := New(Config{
		Output:      &buf,
		Level:       InfoLevel,
		PrettyPrint: &pp,
		NoColor:     true,
		WithCaller:  false,
	})

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			logger.Info().Int("goroutine", n).Msg("concurrent test")
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 100 {
		t.Errorf("expected 100 log lines, got %d", len(lines))
	}
}

// ──────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────

type testError string

func (e testError) Error() string { return string(e) }

func errForTest(msg string) error { return testError(msg) }
