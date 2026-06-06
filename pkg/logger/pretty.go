package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ANSI color codes.
const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiPurple = "\033[35m"
	ansiWhite  = "\033[37m"
	ansiGray   = "\033[90m"
)

// prettyWriter formats JSON log output into human-readable colored lines.
// It is safe for concurrent use.
type prettyWriter struct {
	mu      sync.Mutex
	out     io.Writer
	noColor bool
}

func newPrettyWriter(out io.Writer, noColor bool) *prettyWriter {
	return &prettyWriter{
		out:     out,
		noColor: noColor,
	}
}

// sync.Pool for reusing strings.Builder across calls.
var builderPool = sync.Pool{
	New: func() interface{} {
		b := &strings.Builder{}
		b.Grow(256)
		return b
	},
}

func (w *prettyWriter) Write(p []byte) (int, error) {
	fields := parseOrderedFields(p)
	if fields == nil {
		// Fallback: write raw bytes if JSON parsing fails.
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.out.Write(p)
	}

	var (
		timestamp string
		level     string
		message   string
		caller    string
		errField  string
		extra     []fieldPair
	)

	for _, f := range fields {
		switch f.key {
		case "time":
			timestamp = w.formatTimestamp(f.value)
		case "level":
			level = f.value
		case "message":
			message = f.value
		case "caller":
			caller = formatCaller(f.value)
		case "error":
			errField = f.value
		default:
			extra = append(extra, f)
		}
	}

	sb := builderPool.Get().(*strings.Builder)
	sb.Reset()
	defer builderPool.Put(sb)

	// Timestamp
	if timestamp != "" {
		w.writeColored(sb, ansiGray, timestamp)
		sb.WriteByte(' ')
	}

	// Level badge
	sb.WriteString(w.formatLevel(level))
	sb.WriteByte(' ')

	// Message
	sb.WriteString(message)

	// Error (highlighted)
	if errField != "" {
		sb.WriteByte(' ')
		w.writeColored(sb, ansiRed, "error=")
		w.writeColored(sb, ansiRed, errField)
	}

	// Extra fields
	for _, f := range extra {
		sb.WriteByte(' ')
		w.writeColored(sb, ansiGray, f.key)
		w.writeColored(sb, ansiGray, "=")
		w.writeColored(sb, ansiGreen, f.value)
	}

	// Caller info (at the end, dimmed)
	if caller != "" {
		sb.WriteByte(' ')
		w.writeColored(sb, ansiGray, "("+caller+")")
	}

	sb.WriteByte('\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := io.WriteString(w.out, sb.String())
	if err != nil {
		return n, err
	}
	// Return original byte count to satisfy the writer interface.
	return len(p), nil
}

// writeColored writes text with an ANSI color wrapper, respecting noColor.
func (w *prettyWriter) writeColored(sb *strings.Builder, color, text string) {
	if w.noColor {
		sb.WriteString(text)
		return
	}
	sb.WriteString(color)
	sb.WriteString(text)
	sb.WriteString(ansiReset)
}

// formatLevel returns a colored level badge string.
func (w *prettyWriter) formatLevel(level string) string {
	var color, label string

	switch strings.ToLower(level) {
	case "trace":
		color, label = ansiGray, "TRC"
	case "debug":
		color, label = ansiPurple, "DBG"
	case "info":
		color, label = ansiCyan, "INF"
	case "warn":
		color, label = ansiYellow, "WAR"
	case "error":
		color, label = ansiRed, "ERR"
	case "fatal":
		color, label = ansiRed, "FTL"
	case "panic":
		color, label = ansiRed, "PNC"
	default:
		color = ansiWhite
		label = strings.ToUpper(level)[:3]
	}

	if w.noColor {
		return label
	}
	return color + label + ansiReset
}

// formatTimestamp parses an RFC3339 timestamp and returns a human-friendly UTC string.
func (w *prettyWriter) formatTimestamp(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.UTC().Format("2006/01/02 15:04:05 UTC")
}

// formatCaller shortens a full file path to the last two segments (package/file.go:line).
func formatCaller(caller string) string {
	parts := strings.Split(caller, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return filepath.Base(caller)
}

// ──────────────────────────────────────────────
// Ordered JSON field extraction
// ──────────────────────────────────────────────

// fieldPair holds a key-value pair in the order it appeared in the JSON.
type fieldPair struct {
	key   string
	value string
}

// parseOrderedFields extracts fields from a flat JSON object while preserving key order.
func parseOrderedFields(p []byte) []fieldPair {
	dec := json.NewDecoder(bytes.NewReader(p))

	// Read opening '{'
	tok, err := dec.Token()
	if err != nil {
		return nil
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil
	}

	var fields []fieldPair

	for dec.More() {
		// Read key
		keyTok, err := dec.Token()
		if err != nil {
			break
		}
		key, ok := keyTok.(string)
		if !ok {
			break
		}

		// Read value as raw JSON
		var rawVal json.RawMessage
		if err := dec.Decode(&rawVal); err != nil {
			break
		}

		// Try to unquote as a JSON string first; otherwise use the raw representation.
		var strVal string
		if err := json.Unmarshal(rawVal, &strVal); err != nil {
			strVal = strings.TrimSpace(string(rawVal))
		}

		fields = append(fields, fieldPair{key: key, value: strVal})
	}

	return fields
}
