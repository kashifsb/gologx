package logger

import (
	"strings"
	"unicode/utf8"
)

// Default keys whose values should always be redacted.
var defaultSensitiveKeys = map[string]bool{
	"password":      true,
	"secret":        true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"authorization": true,
	"api_key":       true,
	"apikey":        true,
	"credit_card":   true,
	"ssn":           true,
}

// RedactValue masks a string value, preserving the first and last characters
// for identifiability while hiding the middle.
//
// Examples:
//
//	"mysecrettoken" → "m************n"
//	"ab"            → "**"
//	""              → "****"
func RedactValue(value string) string {
	length := utf8.RuneCountInString(value)
	switch {
	case length == 0:
		return "****"
	case length <= 2:
		return strings.Repeat("*", length)
	case length <= 6:
		runes := []rune(value)
		return string(runes[0]) + strings.Repeat("*", length-1)
	default:
		runes := []rune(value)
		return string(runes[0]) + strings.Repeat("*", length-2) + string(runes[length-1])
	}
}

// RedactedStr adds a string field to an event, automatically redacting
// the value if the key is in the sensitive keys list.
func RedactedStr(event *Event, key, value string) *Event {
	if IsSensitiveKey(key) {
		return event.Str(key, RedactValue(value))
	}
	return event.Str(key, value)
}

// RedactedFields adds multiple fields to an event, redacting any
// whose keys match the sensitive keys list.
func RedactedFields(event *Event, fields map[string]string) *Event {
	for k, v := range fields {
		event = RedactedStr(event, k, v)
	}
	return event
}

// RedactMap takes a map and returns a copy with sensitive values redacted.
// Useful for logging request headers or form data.
func RedactMap(data map[string]string) map[string]string {
	redacted := make(map[string]string, len(data))
	for k, v := range data {
		if IsSensitiveKey(k) {
			redacted[k] = RedactValue(v)
		} else {
			redacted[k] = v
		}
	}
	return redacted
}

// RedactHeaders redacts sensitive HTTP headers.
// It handles the common case where header values are string slices.
func RedactHeaders(headers map[string][]string) map[string]string {
	result := make(map[string]string, len(headers))
	for k, vals := range headers {
		joined := strings.Join(vals, ", ")
		if IsSensitiveKey(k) {
			result[k] = RedactValue(joined)
		} else {
			result[k] = joined
		}
	}
	return result
}

// IsSensitiveKey checks whether a key should be redacted.
// Comparison is case-insensitive and also checks for common prefixes/suffixes.
func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)

	if defaultSensitiveKeys[lower] {
		return true
	}

	// Catch variations like "db_password", "user_token", "x-api-key"
	sensitiveSubstrings := []string{
		"password", "secret", "token", "api_key", "apikey", "api-key", "credential",
	}
	for _, sub := range sensitiveSubstrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}

	return false
}
