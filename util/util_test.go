package util

import (
	"testing"
	"time"
)

func TestCompactDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		maxParts int
		expect   string
	}{
		{5 * time.Second, 2, "5s"},
		{90 * time.Second, 2, "1m30s"},
		{90 * time.Second, 1, "1m"},
		{26 * time.Hour, 2, "1d2h"},
		{26*time.Hour + 10*time.Minute, 3, "1d2h10m"},
	}

	for _, tc := range tests {
		res := CompactDuration(tc.d, tc.maxParts)
		if res != tc.expect {
			t.Errorf("Duration %v, maxParts %d failed: %s != expected %s", tc.d, tc.maxParts, res, tc.expect)
		}
	}
}
