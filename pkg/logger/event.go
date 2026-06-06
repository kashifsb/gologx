package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"
)

// jsonField holds a single JSON key and its already-encoded value.
type jsonField struct {
	key string
	val string // JSON-encoded value
}

// Event represents a log event being built. Methods on Event add fields
// and return the same Event for fluent chaining. Calling Msg or Msgf
// finalizes the event and writes it to the logger's writer.
type Event struct {
	logger          Logger
	level           Level
	fields          []jsonField
	done            bool
	callerSkipExtra int
}

// Str adds a string field.
func (e *Event) Str(key, val string) *Event {
	if e.done {
		return e
	}
	e.fields = append(e.fields, jsonField{key: key, val: strconv.Quote(val)})
	return e
}

// Int adds an int field.
func (e *Event) Int(key string, val int) *Event {
	if e.done {
		return e
	}
	e.fields = append(e.fields, jsonField{key: key, val: strconv.Itoa(val)})
	return e
}

// Int64 adds an int64 field.
func (e *Event) Int64(key string, val int64) *Event {
	if e.done {
		return e
	}
	e.fields = append(e.fields, jsonField{key: key, val: strconv.FormatInt(val, 10)})
	return e
}

// Float64 adds a float64 field.
func (e *Event) Float64(key string, val float64) *Event {
	if e.done {
		return e
	}
	e.fields = append(e.fields, jsonField{key: key, val: strconv.FormatFloat(val, 'f', -1, 64)})
	return e
}

// Bool adds a boolean field.
func (e *Event) Bool(key string, val bool) *Event {
	if e.done {
		return e
	}
	e.fields = append(e.fields, jsonField{key: key, val: strconv.FormatBool(val)})
	return e
}

// Err adds an "error" field from an error value. If err is nil, this is a no-op.
func (e *Event) Err(err error) *Event {
	if e.done || err == nil {
		return e
	}
	e.fields = append(e.fields, jsonField{key: "error", val: strconv.Quote(err.Error())})
	return e
}

// Dur adds a duration field encoded as integer milliseconds.
func (e *Event) Dur(key string, d time.Duration) *Event {
	if e.done {
		return e
	}
	ms := int64(d / time.Millisecond)
	e.fields = append(e.fields, jsonField{key: key, val: strconv.FormatInt(ms, 10)})
	return e
}

// Interface adds a field by marshaling the value to JSON.
func (e *Event) Interface(key string, val interface{}) *Event {
	if e.done {
		return e
	}
	data, err := json.Marshal(val)
	if err != nil {
		data = []byte(strconv.Quote(fmt.Sprintf("%v", val)))
	}
	e.fields = append(e.fields, jsonField{key: key, val: string(data)})
	return e
}

// CallerSkipFrame adds additional caller skip frames.
func (e *Event) CallerSkipFrame(skip int) *Event {
	if e.done {
		return e
	}
	e.callerSkipExtra += skip
	return e
}

// Msg finalizes the event with the given message and writes it.
func (e *Event) Msg(msg string) {
	e.send(msg, 2) // skip: send -> Msg -> caller
}

// Msgf finalizes the event with a formatted message and writes it.
func (e *Event) Msgf(format string, args ...interface{}) {
	e.send(fmt.Sprintf(format, args...), 2) // skip: send -> Msgf -> caller
}

func (e *Event) send(msg string, baseSkip int) {
	if e.done {
		return
	}
	e.done = true

	var buf bytes.Buffer
	buf.Grow(256)

	buf.WriteString(`{"level":`)
	buf.WriteString(strconv.Quote(e.level.String()))

	buf.WriteString(`,"time":`)
	buf.WriteString(strconv.Quote(time.Now().UTC().Format(time.RFC3339)))

	buf.WriteString(`,"message":`)
	buf.WriteString(strconv.Quote(msg))

	// Persistent fields from logger
	for _, f := range e.logger.fields {
		buf.WriteByte(',')
		buf.WriteString(strconv.Quote(f.key))
		buf.WriteByte(':')
		buf.WriteString(f.val)
	}

	// Event-specific fields
	for _, f := range e.fields {
		buf.WriteByte(',')
		buf.WriteString(strconv.Quote(f.key))
		buf.WriteByte(':')
		buf.WriteString(f.val)
	}

	// Caller info
	if e.logger.caller {
		skip := baseSkip + e.callerSkipExtra + e.logger.callerSkip
		_, file, line, ok := runtime.Caller(skip)
		if ok {
			buf.WriteString(`,"caller":`)
			buf.WriteString(strconv.Quote(file + ":" + strconv.Itoa(line)))
		}
	}

	buf.WriteByte('}')
	buf.WriteByte('\n')

	e.logger.mu.Lock()
	e.logger.w.Write(buf.Bytes())
	e.logger.mu.Unlock()

	if e.level == FatalLevel {
		os.Exit(1)
	}
}
