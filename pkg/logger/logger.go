package logger

import (
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config holds all configuration options for the logger.
type Config struct {
	// Output is the destination writer. Defaults to os.Stdout.
	Output io.Writer

	// Level is the minimum log level. Defaults to InfoLevel.
	Level Level

	// PrettyPrint enables human-readable colored output.
	// When false, logs are emitted as raw JSON (suitable for production).
	// If nil (default), auto-detected based on terminal attachment.
	PrettyPrint *bool

	// NoColor disables ANSI color codes in pretty-print mode.
	// Automatically set if NO_COLOR env var is present or TERM=dumb.
	NoColor bool

	// WithCaller includes source file and line number in every log entry.
	WithCaller bool

	// CallerSkipFrames adjusts the caller skip depth.
	// Useful if you wrap the logger in additional layers.
	CallerSkipFrames int

	// ServiceName is an optional field attached to every log entry.
	ServiceName string
}

// Logger is a thread-safe structured logger.
type Logger struct {
	w          io.Writer
	level      Level
	fields     []jsonField
	caller     bool
	callerSkip int
	mu         *sync.Mutex
}

// defaultLogger is the package-level logger used as a fallback by FromContext.
var defaultLogger = Logger{
	w:     os.Stdout,
	level: InfoLevel,
	mu:    &sync.Mutex{},
}

// New creates a Logger from the given Config.
// It does NOT modify any global state.
func New(cfg Config) Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	prettyPrint := cfg.shouldPrettyPrint()
	noColor := cfg.shouldDisableColor()

	var writer io.Writer
	if prettyPrint {
		writer = newPrettyWriter(cfg.Output, noColor)
	} else {
		// Raw JSON — ideal for Datadog, ELK, Loki, CloudWatch, etc.
		writer = cfg.Output
	}

	l := Logger{
		w:     writer,
		level: cfg.Level,
		mu:    &sync.Mutex{},
	}

	if cfg.WithCaller {
		l.caller = true
		l.callerSkip = cfg.CallerSkipFrames
	}

	if cfg.ServiceName != "" {
		l.fields = append(l.fields, jsonField{
			key: "service",
			val: strconv.Quote(cfg.ServiceName),
		})
	}

	return l
}

// InitGlobal creates a logger from cfg and sets it as the default package logger.
// This is a convenience for applications that prefer the global log.Info() style.
func InitGlobal(cfg Config) {
	defaultLogger = New(cfg)
}

// InitGlobalFromEnv configures the default logger using environment variables:
//
//	LOG_LEVEL    — trace, debug, info (default), warn, error, fatal
//	LOG_FORMAT   — json, pretty (auto-detected if unset)
//	LOG_CALLER   — true/false (default: true)
//	NO_COLOR     — if set, disables color output
//	SERVICE_NAME — optional service identifier
func InitGlobalFromEnv() {
	cfg := Config{
		Output:     os.Stdout,
		Level:      parseLevelFromEnv(),
		NoColor:    shouldDisableColorFromEnv(),
		WithCaller: parseCallerFromEnv(),
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		switch strings.ToLower(format) {
		case "json":
			pp := false
			cfg.PrettyPrint = &pp
		case "pretty":
			pp := true
			cfg.PrettyPrint = &pp
		}
	}

	if svc := os.Getenv("SERVICE_NAME"); svc != "" {
		cfg.ServiceName = svc
	}

	InitGlobal(cfg)
}

// Trace returns a new Event at trace level.
func (l Logger) Trace() *Event { return l.newEvent(TraceLevel) }

// Debug returns a new Event at debug level.
func (l Logger) Debug() *Event { return l.newEvent(DebugLevel) }

// Info returns a new Event at info level.
func (l Logger) Info() *Event { return l.newEvent(InfoLevel) }

// Warn returns a new Event at warn level.
func (l Logger) Warn() *Event { return l.newEvent(WarnLevel) }

// Error returns a new Event at error level.
func (l Logger) Error() *Event { return l.newEvent(ErrorLevel) }

// Fatal returns a new Event at fatal level.
func (l Logger) Fatal() *Event { return l.newEvent(FatalLevel) }

func (l Logger) newEvent(level Level) *Event {
	if level < l.level {
		return &Event{done: true}
	}
	return &Event{logger: l, level: level}
}

// With returns a Context for building a sub-logger with additional fields.
func (l Logger) With() Context {
	// Deep-copy the fields so mutations don't affect the original.
	fields := make([]jsonField, len(l.fields))
	copy(fields, l.fields)
	l.fields = fields
	return Context{l: l}
}

// shouldPrettyPrint determines whether to use pretty-printed output.
func (c *Config) shouldPrettyPrint() bool {
	if c.PrettyPrint != nil {
		return *c.PrettyPrint
	}
	return isTerminal(c.Output)
}

// shouldDisableColor checks if color should be turned off.
func (c *Config) shouldDisableColor() bool {
	if c.NoColor {
		return true
	}
	return shouldDisableColorFromEnv()
}

// shouldDisableColorFromEnv respects the NO_COLOR (https://no-color.org/) convention.
func shouldDisableColorFromEnv() bool {
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return true
	}
	return os.Getenv("TERM") == "dumb"
}

// isTerminal checks whether the writer is attached to an interactive terminal.
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err == nil {
			return fi.Mode()&os.ModeCharDevice != 0
		}
	}
	return false
}

// parseLevelFromEnv reads LOG_LEVEL from the environment.
func parseLevelFromEnv() Level {
	return ParseLevel(os.Getenv("LOG_LEVEL"))
}

// parseCallerFromEnv reads LOG_CALLER from the environment.
// Defaults to true if unset.
func parseCallerFromEnv() bool {
	val := strings.ToLower(os.Getenv("LOG_CALLER"))
	if val == "false" || val == "0" || val == "no" {
		return false
	}
	return true
}
