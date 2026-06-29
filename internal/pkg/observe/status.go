package observe

// StatusClass 将 HTTP 状态码归类到有限的 status label 集合（2xx/3xx/4xx/5xx/unknown），
// 避免 Prometheus label 基数爆炸。HTTP 中间件与 infra/metrics 共用此函数保持一致。
func StatusClass(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}
