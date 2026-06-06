package logger

import "strconv"

// Context is used to build sub-loggers with additional persistent fields.
type Context struct {
	l Logger
}

// Str adds a persistent string field to the sub-logger.
func (c Context) Str(key, val string) Context {
	c.l.fields = append(c.l.fields, jsonField{key: key, val: strconv.Quote(val)})
	return c
}

// CallerWithSkipFrameCount enables caller reporting and sets the skip frame count.
func (c Context) CallerWithSkipFrameCount(skip int) Context {
	c.l.caller = true
	c.l.callerSkip = skip
	return c
}

// Timestamp is a no-op (timestamps are always included by Event.send).
func (c Context) Timestamp() Context {
	return c
}

// Logger returns the configured Logger.
func (c Context) Logger() Logger {
	return c.l
}
