package logger

import "strings"

// Level defines the severity of a log message.
type Level int8

const (
	TraceLevel Level = -1
	DebugLevel Level = 0
	InfoLevel  Level = 1
	WarnLevel  Level = 2
	ErrorLevel Level = 3
	FatalLevel Level = 4
	Disabled   Level = 7
)

// String returns the lowercase name for this level.
func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return ""
	}
}

// ParseLevel converts a level name to a Level constant.
// Unrecognized strings default to InfoLevel.
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "trace":
		return TraceLevel
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}
