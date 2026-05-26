package rpchandler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// checkDataAge returns an error when maxDataAge > 0 and the data is older than that limit.
// Pass maxDataAge = 0 to disable the check entirely.
func checkDataAge(dataTime time.Time, maxDataAge time.Duration) error {
	if maxDataAge <= 0 {
		return nil
	}
	age := time.Since(dataTime)
	if age > maxDataAge {
		return fmt.Errorf("data age %s exceeds maximum allowed %s", age.Round(time.Second), maxDataAge)
	}
	return nil
}

// parseTimestampParam extracts a time.Time from JSON data using a dot-separated path and an optional
// format hint ("unix", "unix_ms", or a Go time layout string; defaults to RFC3339/RFC3339Nano).
// Returns the zero Time and a non-nil error if the path is empty, the field is missing, or parsing fails.
func parseTimestampParam(data map[string]any, dotPath, format string) (time.Time, error) {
	if dotPath == "" {
		return time.Time{}, fmt.Errorf("timestamp_path is empty")
	}
	parts := strings.Split(dotPath, ".")
	var cur any = data
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return time.Time{}, fmt.Errorf("expected object at %q, got %T", p, cur)
		}
		cur, ok = m[p]
		if !ok {
			return time.Time{}, fmt.Errorf("key %q not found", p)
		}
	}

	switch strings.ToLower(format) {
	case "unix":
		return parseUnixTimestamp(cur, 1)
	case "unix_ms":
		return parseUnixTimestamp(cur, 1000)
	default:
		// Treat as a time layout string or RFC3339 if empty.
		layout := format
		if layout == "" {
			return parseRFC3339(cur)
		}
		s, err := anyToString(cur)
		if err != nil {
			return time.Time{}, err
		}
		t, err := time.Parse(layout, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse timestamp %q with layout %q: %w", s, layout, err)
		}
		return t, nil
	}
}

func parseUnixTimestamp(v any, divisor int64) (time.Time, error) {
	var secs int64
	switch x := v.(type) {
	case float64:
		secs = int64(x) / divisor
	case int64:
		secs = x / divisor
	case int:
		secs = int64(x) / divisor
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse unix timestamp string %q: %w", x, err)
		}
		secs = n / divisor
	default:
		return time.Time{}, fmt.Errorf("unexpected timestamp type %T", v)
	}
	return time.Unix(secs, 0), nil
}

func parseRFC3339(v any) (time.Time, error) {
	s, err := anyToString(v)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("parse RFC3339 timestamp %q: %w", s, err)
	}
	return t, nil
}

func anyToString(v any) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string timestamp, got %T", v)
	}
	return strings.TrimSpace(s), nil
}
