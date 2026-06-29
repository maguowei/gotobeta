package httpclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/maguowei/gotobeta/internal/infra/config"
	"github.com/maguowei/gotobeta/internal/infra/metrics"
)

// New 为指定 target 创建 Resty client。
func New(cfg config.HTTPClientConfig, targetName string, mc *metrics.Collectors) (*resty.Client, error) {
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return nil, fmt.Errorf("http client target name is required")
	}
	target, ok := cfg.Targets[targetName]
	if !ok {
		return nil, fmt.Errorf("http client target %q not configured", targetName)
	}
	if strings.TrimSpace(target.BaseURL) == "" {
		return nil, fmt.Errorf("http client target %q base_url is required", targetName)
	}

	timeout, err := targetTimeout(cfg, target)
	if err != nil {
		return nil, fmt.Errorf("http client target %q timeout: %w", targetName, err)
	}

	transport := observedTransport{
		target:  targetName,
		metrics: mc,
		next:    otelhttp.NewTransport(defaultTransport()),
	}
	client := resty.NewWithClient(&http.Client{
		Timeout:   timeout,
		Transport: transport,
	})
	client.SetBaseURL(strings.TrimRight(target.BaseURL, "/"))
	if limit := responseBodyLimit(cfg, target); limit > 0 {
		client.SetResponseBodyLimit(limit)
	}
	return client, nil
}

func targetTimeout(cfg config.HTTPClientConfig, target config.HTTPClientTarget) (time.Duration, error) {
	value := strings.TrimSpace(target.Timeout)
	if value == "" {
		value = cfg.DefaultTimeout
	}
	return time.ParseDuration(value)
}

func responseBodyLimit(cfg config.HTTPClientConfig, target config.HTTPClientTarget) int {
	if target.ResponseBodyLimit > 0 {
		return int(target.ResponseBodyLimit)
	}
	return int(cfg.DefaultResponseBodyLimit)
}

func defaultTransport() *http.Transport {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DialContext = (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	base.MaxIdleConns = 100
	base.MaxIdleConnsPerHost = 10
	base.IdleConnTimeout = 90 * time.Second
	base.TLSHandshakeTimeout = 10 * time.Second
	base.ExpectContinueTimeout = time.Second
	return base
}

type observedTransport struct {
	target  string
	metrics *metrics.Collectors
	next    http.RoundTripper
}

func (t observedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.next.RoundTrip(req)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	operation := strings.ToLower(req.Method)
	if operation == "" {
		operation = "request"
	}
	t.metrics.ObserveExternalCall(req.Context(), t.target, operation, time.Since(start), statusCode, contextError(req.Context(), err))
	return resp, err
}

func contextError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return err
}
