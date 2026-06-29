package todo

import "testing"

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusPending, true},
		{StatusDone, true},
		{Status("unknown"), false},
	}
	for _, tt := range tests {
		if got := tt.status.IsValid(); got != tt.want {
			t.Errorf("IsValid(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}
