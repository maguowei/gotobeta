package timeutil

import (
	"testing"
	"time"
)

func TestFormatTime(t *testing.T) {
	ts := time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC)
	got := FormatTime(ts)
	want := "2026-01-02 15:04:05"
	if got != want {
		t.Errorf("FormatTime() = %q, want %q", got, want)
	}
}
