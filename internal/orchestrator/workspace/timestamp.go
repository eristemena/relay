package workspace

import (
	"strings"
	"time"
)

func formatEventTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseEventTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
}

func timestampHasSubsecondPrecision(value string) bool {
	return strings.Contains(strings.TrimSpace(value), ".")
}
