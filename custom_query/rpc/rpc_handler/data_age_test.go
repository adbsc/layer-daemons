package rpchandler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCheckDataAge_disabled(t *testing.T) {
	// maxDataAge == 0 always passes regardless of how old the data is.
	ancient := time.Now().Add(-24 * time.Hour)
	require.NoError(t, checkDataAge(ancient, 0))
}

func TestCheckDataAge_fresh(t *testing.T) {
	recent := time.Now().Add(-30 * time.Second)
	require.NoError(t, checkDataAge(recent, 5*time.Minute))
}

func TestCheckDataAge_stale(t *testing.T) {
	old := time.Now().Add(-10 * time.Minute)
	err := checkDataAge(old, 5*time.Minute)
	require.Error(t, err)
	require.Contains(t, err.Error(), "data age")
	require.Contains(t, err.Error(), "exceeds maximum")
}

func TestCheckDataAge_exactlyAtLimit(t *testing.T) {
	// Exactly at the limit should still pass (age <= maxAge).
	atLimit := time.Now().Add(-5 * time.Minute)
	// Allow a small margin for test execution time.
	require.NoError(t, checkDataAge(atLimit, 5*time.Minute+time.Second))
}

// parseTimestampParam tests

func TestParseTimestampParam_unix(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	data := map[string]any{"ts": float64(now.Unix())}
	got, err := parseTimestampParam(data, "ts", "unix")
	require.NoError(t, err)
	require.Equal(t, now, got)
}

func TestParseTimestampParam_unix_ms(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	data := map[string]any{"ts": float64(now.UnixMilli())}
	got, err := parseTimestampParam(data, "ts", "unix_ms")
	require.NoError(t, err)
	require.Equal(t, now, got)
}

func TestParseTimestampParam_rfc3339(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	data := map[string]any{"updated": now.Format(time.RFC3339)}
	got, err := parseTimestampParam(data, "updated", "")
	require.NoError(t, err)
	require.True(t, now.Equal(got), "expected %v, got %v", now, got)
}

func TestParseTimestampParam_nestedPath(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	data := map[string]any{
		"meta": map[string]any{
			"block": map[string]any{
				"timestamp": float64(now.Unix()),
			},
		},
	}
	got, err := parseTimestampParam(data, "meta.block.timestamp", "unix")
	require.NoError(t, err)
	require.Equal(t, now, got)
}

func TestParseTimestampParam_missingKey(t *testing.T) {
	data := map[string]any{"other": "value"}
	_, err := parseTimestampParam(data, "ts", "unix")
	require.Error(t, err)
}

func TestParseTimestampParam_emptyPath(t *testing.T) {
	_, err := parseTimestampParam(map[string]any{}, "", "unix")
	require.Error(t, err)
}
