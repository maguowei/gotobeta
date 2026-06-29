package ws

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginCheckerWhitelist(t *testing.T) {
	check := originChecker([]string{"https://app.example.com"})

	allowed := httptest.NewRequest(http.MethodGet, "/ws", nil)
	allowed.Header.Set("Origin", "https://app.example.com")
	if !check(allowed) {
		t.Fatal("白名单内 Origin 应放行")
	}

	denied := httptest.NewRequest(http.MethodGet, "/ws", nil)
	denied.Header.Set("Origin", "https://evil.example.com")
	if check(denied) {
		t.Fatal("白名单外 Origin 应拒绝")
	}
}

func TestOriginCheckerAllowsEmptyOrigin(t *testing.T) {
	// 原生客户端（非浏览器）通常不带 Origin 头，应放行（鉴权由 ticket 兜底）。
	check := originChecker([]string{"https://app.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	if !check(req) {
		t.Fatal("无 Origin 头应放行")
	}
}

func TestOriginCheckerEmptyWhitelistSameOrigin(t *testing.T) {
	// 未配置白名单时仅放行同源，拒绝跨域。
	check := originChecker(nil)

	same := httptest.NewRequest(http.MethodGet, "/ws", nil)
	same.Host = "im.example.com"
	same.Header.Set("Origin", "https://im.example.com")
	if !check(same) {
		t.Fatal("同源应放行")
	}

	cross := httptest.NewRequest(http.MethodGet, "/ws", nil)
	cross.Host = "im.example.com"
	cross.Header.Set("Origin", "https://other.example.com")
	if check(cross) {
		t.Fatal("跨域且无白名单应拒绝")
	}
}
