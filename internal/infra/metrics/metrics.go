package metrics

import (
	"context"
	"errors"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/observe"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Collectors 持有所有 Prometheus 指标收集器。
// 通过 NewCollectors 创建并由组合根注入到中间件和 infra 层，不使用全局状态。
type Collectors struct {
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	ExternalCallTotal    *prometheus.CounterVec
	ExternalCallDuration *prometheus.HistogramVec
	EventBusEventsTotal  *prometheus.CounterVec
	EventBusDuration     *prometheus.HistogramVec

	// IM 关键指标
	WSConnectionsActive prometheus.Gauge     // 当前活跃 WS 连接数
	MessageE2ELatency   prometheus.Histogram // 消息端到端延迟（发送到投递）
	SeqAllocDuration    prometheus.Histogram // 每会话 seq 分配耗时
	PushTotal           *prometheus.CounterVec
}

// NewCollectors 创建并注册指标收集器。
// registry 由组合根注入（每个进程独立），不使用全局 DefaultRegisterer。
func NewCollectors(registry prometheus.Registerer, namespace string) *Collectors {
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)

	externalCallTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "external_call_total",
			Help:      "Total number of outbound calls to external services",
		},
		[]string{"client", "operation", "status"},
	)

	externalCallDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "external_call_duration_seconds",
			Help:      "External call duration in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"client", "operation"},
	)

	eventBusEventsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "eventbus_events_total",
			Help:      "Total number of inbox/outbox events processed",
		},
		[]string{"component", "event_type", "status"},
	)

	eventBusDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "eventbus_duration_seconds",
			Help:      "Inbox/outbox event processing duration in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"component", "event_type"},
	)

	wsConnectionsActive := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ws_connections_active",
			Help:      "Current number of active WebSocket connections",
		},
	)

	messageE2ELatency := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_e2e_latency_seconds",
			Help:      "End-to-end IM message latency from send to delivery in seconds",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
	)

	seqAllocDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "seq_alloc_duration_seconds",
			Help:      "Per-conversation sequence allocation duration in seconds",
			Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
	)

	pushTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "push_total",
			Help:      "Total number of realtime push attempts by result",
		},
		[]string{"result"},
	)

	registry.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		externalCallTotal,
		externalCallDuration,
		eventBusEventsTotal,
		eventBusDuration,
		wsConnectionsActive,
		messageE2ELatency,
		seqAllocDuration,
		pushTotal,
	)

	return &Collectors{
		HTTPRequestsTotal:    httpRequestsTotal,
		HTTPRequestDuration:  httpRequestDuration,
		ExternalCallTotal:    externalCallTotal,
		ExternalCallDuration: externalCallDuration,
		EventBusEventsTotal:  eventBusEventsTotal,
		EventBusDuration:     eventBusDuration,
		WSConnectionsActive:  wsConnectionsActive,
		MessageE2ELatency:    messageE2ELatency,
		SeqAllocDuration:     seqAllocDuration,
		PushTotal:            pushTotal,
	}
}

// ClassifyHTTPStatus 将外部调用结果归类到有限 status label。
func ClassifyHTTPStatus(statusCode int, err error) string {
	if statusCode > 0 {
		return observe.StatusClass(statusCode)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	return "network_error"
}

// ObserveExternalCall 记录一次外部调用。
// nil receiver 安全：metrics 是可选注入的依赖，未注入时静默跳过。
func (c *Collectors) ObserveExternalCall(ctx context.Context, client string, operation string, duration time.Duration, statusCode int, err error) {
	if c == nil {
		return
	}
	status := ClassifyHTTPStatus(statusCode, err)
	c.ExternalCallTotal.WithLabelValues(client, operation, status).Inc()

	observer := c.ExternalCallDuration.WithLabelValues(client, operation)
	observe.ObserveWithTraceID(ctx, observer, duration.Seconds())
}

// ObserveEventBus 记录一次 inbox/outbox 处理结果。
// nil receiver 安全：metrics 是可选注入的依赖（WithInboxMetrics/WithOutboxMetrics），未注入时静默跳过。
func (c *Collectors) ObserveEventBus(ctx context.Context, component string, eventType string, status string, duration time.Duration) {
	if c == nil {
		return
	}
	c.EventBusEventsTotal.WithLabelValues(component, eventType, status).Inc()

	observer := c.EventBusDuration.WithLabelValues(component, eventType)
	observe.ObserveWithTraceID(ctx, observer, duration.Seconds())
}

// ObserveSeqAlloc 记录一次每会话 seq 分配耗时。nil receiver 安全。
func (c *Collectors) ObserveSeqAlloc(ctx context.Context, duration time.Duration) {
	if c == nil {
		return
	}
	observe.ObserveWithTraceID(ctx, c.SeqAllocDuration, duration.Seconds())
}

// ObserveMessageLatency 记录一次消息端到端延迟。nil receiver 安全。
func (c *Collectors) ObserveMessageLatency(ctx context.Context, duration time.Duration) {
	if c == nil {
		return
	}
	observe.ObserveWithTraceID(ctx, c.MessageE2ELatency, duration.Seconds())
}

// IncPush 记录一次实时推送结果（result 为 success/dropped/error 等有限标签）。nil receiver 安全。
func (c *Collectors) IncPush(result string) {
	if c == nil {
		return
	}
	c.PushTotal.WithLabelValues(result).Inc()
}

// SetWSConnections 设置当前活跃 WS 连接数。nil receiver 安全。
func (c *Collectors) SetWSConnections(n float64) {
	if c == nil {
		return
	}
	c.WSConnectionsActive.Set(n)
}
