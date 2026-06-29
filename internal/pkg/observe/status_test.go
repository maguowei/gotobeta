package observe

import "testing"

func TestStatusClass(t *testing.T) {
	t.Parallel()
	tests := []struct {
		code int
		want string
	}{
		{200, "2xx"},
		{204, "2xx"},
		{301, "3xx"},
		{400, "4xx"},
		{404, "4xx"},
		{500, "5xx"},
		{503, "5xx"},
		{0, "unknown"},
		{700, "unknown"},
	}
	for _, tt := range tests {
		if got := StatusClass(tt.code); got != tt.want {
			t.Fatalf("StatusClass(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
