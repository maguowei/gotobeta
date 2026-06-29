package sensitive

import "testing"

func TestRedactPhone(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":            "",
		"13800138000": "138****8000",
		"123":         "***",
		"中文1380013xx": "中文1****13xx",
	}
	for in, want := range cases {
		if got := RedactPhone(in); got != want {
			t.Errorf("RedactPhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRedactEmail(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"":                  "",
		"a@b.com":           "a***@b.com",
		"alice@example.com": "a***@example.com",
		"@x.com":            "******",
	}
	for in, want := range cases {
		if got := RedactEmail(in); got != want {
			t.Errorf("RedactEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRedactToken(t *testing.T) {
	t.Parallel()
	if got := RedactToken("short"); got != "*****" {
		t.Fatalf("short = %q", got)
	}
	if got := RedactToken("0123456789abcdef"); got != "0123****cdef" {
		t.Fatalf("long = %q", got)
	}
}

func TestRedactIDCard(t *testing.T) {
	t.Parallel()
	if got := RedactIDCard("123456789012345678"); got != "1234**********5678" {
		t.Fatalf("id card = %q", got)
	}
	if got := RedactIDCard("1234567"); got != "*******" {
		t.Fatalf("short id card = %q", got)
	}
}

func TestRedactBearer(t *testing.T) {
	t.Parallel()
	if got := RedactBearer("Bearer 0123456789abcdef"); got != "Bearer 0123****cdef" {
		t.Fatalf("got = %q", got)
	}
	if got := RedactBearer("0123456789abcdef"); got != "0123****cdef" {
		t.Fatalf("no-prefix got = %q", got)
	}
}

func TestMaskByContentType(t *testing.T) {
	t.Parallel()

	got := MaskByContentType("application/json", []byte(`{"access_token":"secret","nested":{"secret_key":"value"},"name":"demo"}`))
	if got != `{"access_token":"***","name":"demo","nested":{"secret_key":"***"}}` {
		t.Fatalf("masked json = %q", got)
	}

	got = MaskByContentType("application/json", []byte(`{"password":"secret"`))
	if got == `{"password":"secret"` || got != "[invalid json body, size=20]" {
		t.Fatalf("malformed json mask = %q", got)
	}

	got = MaskByContentType("application/x-www-form-urlencoded", []byte(`username=alice&password=secret&token=abc`))
	if got != "password=%2A%2A%2A&token=%2A%2A%2A&username=alice" {
		t.Fatalf("masked form = %q", got)
	}

	got = MaskByContentType("bad content type", []byte(`password=secret`))
	if got != "[invalid json body, size=15]" {
		t.Fatalf("fallback mask = %q", got)
	}

	got = MaskByContentType("text/plain", nil)
	if got != "" {
		t.Fatalf("empty body mask = %q", got)
	}
}
