package server

import (
	"net/http"
	"testing"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

// TestServeHTTPSelectsTLSBranch 验证 serveHTTP 按 TLS 配置选择监听方式：
// tls.enabled 时走 ListenAndServeTLS（带证书），否则走明文 HTTP。
func TestServeHTTPSelectsTLSBranch(t *testing.T) {
	origHTTP, origTLS := listenAndServeHTTP, listenAndServeTLS
	t.Cleanup(func() { listenAndServeHTTP, listenAndServeTLS = origHTTP, origTLS })

	t.Run("tls enabled uses TLS listener", func(t *testing.T) {
		var httpCalled bool
		var gotCert, gotKey string
		listenAndServeHTTP = func(*http.Server) error { httpCalled = true; return nil }
		listenAndServeTLS = func(_ *http.Server, cert, key string) error {
			gotCert, gotKey = cert, key
			return nil
		}

		err := serveHTTP(&http.Server{}, config.TLSConfig{
			Enabled: true, CertFile: "/c/tls.crt", KeyFile: "/c/tls.key",
		})
		if err != nil {
			t.Fatalf("serveHTTP error = %v", err)
		}
		if httpCalled {
			t.Fatal("tls.enabled 时不应调用明文 HTTP 监听")
		}
		if gotCert != "/c/tls.crt" || gotKey != "/c/tls.key" {
			t.Fatalf("TLS 监听证书参数 = %q,%q", gotCert, gotKey)
		}
	})

	t.Run("tls disabled uses plain listener", func(t *testing.T) {
		var httpCalled, tlsCalled bool
		listenAndServeHTTP = func(*http.Server) error { httpCalled = true; return nil }
		listenAndServeTLS = func(*http.Server, string, string) error { tlsCalled = true; return nil }

		if err := serveHTTP(&http.Server{}, config.TLSConfig{Enabled: false}); err != nil {
			t.Fatalf("serveHTTP error = %v", err)
		}
		if !httpCalled {
			t.Fatal("tls 未启用时应调用明文 HTTP 监听")
		}
		if tlsCalled {
			t.Fatal("tls 未启用时不应调用 TLS 监听")
		}
	})
}
