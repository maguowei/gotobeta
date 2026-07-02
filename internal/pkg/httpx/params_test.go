package httpx

import "testing"

func TestParsePositiveID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    int64
		wantErr bool
	}{
		{name: "valid", raw: "42", want: 42},
		{name: "not a number", raw: "abc", wantErr: true},
		{name: "zero", raw: "0", wantErr: true},
		{name: "negative", raw: "-1", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParsePositiveID(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParsePositiveID(%q) error = nil, want error", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePositiveID(%q) error = %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("ParsePositiveID(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}
