package httpclient

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/maguowei/gotobeta/internal/infra/config"
)

func TestNewUsesRestyWithConfiguredHTTPClient(t *testing.T) {
	cfg := config.HTTPClientConfig{
		DefaultTimeout:           "5s",
		DefaultResponseBodyLimit: 1024,
		Targets: map[string]config.HTTPClientTarget{
			"billing": {
				BaseURL:           "https://example.com",
				Timeout:           "2s",
				ResponseBodyLimit: 256,
			},
		},
	}

	client, err := New(cfg, "billing", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, ok := any(client).(*resty.Client); !ok {
		t.Fatalf("New() should return *resty.Client")
	}
	if client.GetClient().Timeout != 2*time.Second {
		t.Fatalf("timeout = %v, want 2s", client.GetClient().Timeout)
	}
	if client.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL = %q", client.BaseURL)
	}
	if client.ResponseBodyLimit != 256 {
		t.Fatalf("ResponseBodyLimit = %d, want 256", client.ResponseBodyLimit)
	}
}

func TestNewUsesObservedOtelTransport(t *testing.T) {
	cfg := config.HTTPClientConfig{
		DefaultTimeout:           "5s",
		DefaultResponseBodyLimit: 1024,
		Targets: map[string]config.HTTPClientTarget{
			"billing": {BaseURL: "https://example.com"},
		},
	}

	client, err := New(cfg, "billing", nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	transport, ok := client.GetClient().Transport.(observedTransport)
	if !ok {
		t.Fatalf("transport = %T, want observedTransport", client.GetClient().Transport)
	}
	if transport.target != "billing" {
		t.Fatalf("target = %q, want billing", transport.target)
	}
	if got := fmt.Sprintf("%T", transport.next); !strings.Contains(got, "otelhttp") {
		t.Fatalf("inner transport = %s, want otelhttp transport", got)
	}
}

func TestNewRejectsMissingTarget(t *testing.T) {
	_, err := New(config.HTTPClientConfig{DefaultTimeout: "5s"}, "billing", nil)
	if err == nil {
		t.Fatal("New() error = nil, want missing target error")
	}
}

func TestObservedTransportRecordsBoundedStatus(t *testing.T) {
	transport := observedTransport{
		target: "billing",
		next: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Body: http.NoBody}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "https://example.com/invoices/123", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", resp.StatusCode)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
