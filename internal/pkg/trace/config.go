package trace

// Config 是分布式追踪配置。
// Endpoint 非空即视为启用：只有一个真值源，避免 enabled/endpoint 语义冲突。
type Config struct {
	Endpoint       string  `mapstructure:"endpoint"`
	Insecure       bool    `mapstructure:"insecure"`
	Sampler        string  `mapstructure:"sampler"`
	SampleRatio    float64 `mapstructure:"sample_ratio"`
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
}

// ShouldExport 判断是否实际启用 OTLP exporter。
// Endpoint 非空即视为启用。
func (c Config) ShouldExport() bool {
	return c.Endpoint != ""
}
