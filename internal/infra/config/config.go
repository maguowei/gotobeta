package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"time"

	"github.com/spf13/viper"
)

// Config 是应用配置。
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Logger  LoggerConfig  `mapstructure:"logger"`
	Audit   AuditConfig   `mapstructure:"audit"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Sentry  SentryConfig  `mapstructure:"sentry"`
	Tracing TracingConfig `mapstructure:"tracing"`

	HTTPClient HTTPClientConfig `mapstructure:"http_client"`

	Redis RedisConfig `mapstructure:"redis"`

	SMTP SMTPConfig `mapstructure:"smtp"`

	Auth AuthConfig `mapstructure:"auth"`

	Database DatabaseConfig `mapstructure:"database"`

	IM IMConfig `mapstructure:"im"`

	ObjStore ObjStoreConfig `mapstructure:"objstore"`

	LoadedFiles []string `mapstructure:"-"`
}

// IMConfig 是即时通讯相关配置。
type IMConfig struct {
	RecallWindow    string `mapstructure:"recall_window"`     // 撤回时间窗口，如 2m
	PresenceTTL     string `mapstructure:"presence_ttl"`      // 在线状态 Redis TTL，如 30s
	WSTicketTTL     string `mapstructure:"ws_ticket_ttl"`     // WS 一次性 ticket TTL，如 30s
	MessagePageSize int    `mapstructure:"message_page_size"` // 增量拉取默认页大小

	MessageRatePerMinute int `mapstructure:"message_rate_per_minute"` // 单用户发消息稳态速率（条/分钟）
	MessageRateBurst     int `mapstructure:"message_rate_burst"`      // 单用户发消息突发容量
}

// ObjStoreConfig 是 S3 兼容对象存储配置。dev 指向 MinIO，prod 指向任意 S3 兼容存储。
type ObjStoreConfig struct {
	Endpoint      string `mapstructure:"endpoint"`        // 形如 127.0.0.1:9000（不含协议）
	Region        string `mapstructure:"region"`          // 区域，可空
	Bucket        string `mapstructure:"bucket"`          // 桶名
	AccessKey     string `mapstructure:"access_key"`      // 访问密钥 ID
	SecretKey     string `mapstructure:"secret_key"`      // 访问密钥
	UseSSL        bool   `mapstructure:"use_ssl"`         // 是否 https
	PublicBaseURL string `mapstructure:"public_base_url"` // 对外访问基址，可空
	PresignTTL    string `mapstructure:"presign_ttl"`     // 预签名有效期，如 15m
}

// ServerConfig 是 HTTP 服务配置。
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// LoggerConfig 是日志配置。
type LoggerConfig struct {
	Level   string `mapstructure:"level"`
	Path    string `mapstructure:"path"`
	AppName string `mapstructure:"app_name"`
	AppEnv  string `mapstructure:"app_env"`
}

// AuditConfig 是审计日志配置。
type AuditConfig struct {
	Enabled             bool `mapstructure:"enabled"`
	LogRequestBody      bool `mapstructure:"log_request_body"`
	LogResponseBody     bool `mapstructure:"log_response_body"`
	MaskSensitiveFields bool `mapstructure:"mask_sensitive_fields"`
	MaxBodyBytes        int  `mapstructure:"max_body_bytes"`
}

// MetricsConfig 是指标配置。
type MetricsConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Path      string `mapstructure:"path"`
	Namespace string `mapstructure:"namespace"`
}

// SentryConfig 是 Sentry 配置。
type SentryConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DSN     string `mapstructure:"dsn"`
	Env     string `mapstructure:"env"`
}

// TracingConfig 是分布式追踪配置。
// Endpoint 非空即视为启用：只有一个真值源，避免 enabled/endpoint 语义冲突。
type TracingConfig struct {
	Endpoint       string  `mapstructure:"endpoint"`
	Insecure       bool    `mapstructure:"insecure"`
	Sampler        string  `mapstructure:"sampler"`
	SampleRatio    float64 `mapstructure:"sample_ratio"`
	ServiceName    string  `mapstructure:"service_name"`
	ServiceVersion string  `mapstructure:"service_version"`
}

// HTTPClientConfig 是出站 HTTP client 配置。
type HTTPClientConfig struct {
	DefaultTimeout           string                      `mapstructure:"default_timeout"`
	DefaultResponseBodyLimit int64                       `mapstructure:"default_response_body_limit"`
	Targets                  map[string]HTTPClientTarget `mapstructure:"targets"`
}

// HTTPClientTarget 是单个外部 HTTP target 的配置。
type HTTPClientTarget struct {
	BaseURL           string `mapstructure:"base_url"`
	Timeout           string `mapstructure:"timeout"`
	ResponseBodyLimit int64  `mapstructure:"response_body_limit"`
}

// RedisConfig 是 Redis cache client 配置。
type RedisConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	DialTimeout  string `mapstructure:"dial_timeout"`
	ReadTimeout  string `mapstructure:"read_timeout"`
	WriteTimeout string `mapstructure:"write_timeout"`
	KeyPrefix    string `mapstructure:"key_prefix"`
}

// SMTPConfig 是 SMTP 邮件发送配置。
type SMTPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	TLSMode  string `mapstructure:"tls_mode"`
	Timeout  string `mapstructure:"timeout"`
}

// AuthConfig 是认证配置。
type AuthConfig struct {
	JWT AuthJWTConfig `mapstructure:"jwt"`

	OAuth     AuthOAuthConfig     `mapstructure:"oauth"`
	Email     AuthEmailConfig     `mapstructure:"email"`
	RateLimit AuthRateLimitConfig `mapstructure:"rate_limit"`
}

// AuthJWTConfig 是 JWT Bearer 认证配置。
type AuthJWTConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Issuer     string `mapstructure:"issuer"`
	Audience   string `mapstructure:"audience"`
	HMACSecret string `mapstructure:"hmac_secret"`
	ClockSkew  string `mapstructure:"clock_skew"`

	AccessTTL  string `mapstructure:"access_ttl"`
	RefreshTTL string `mapstructure:"refresh_ttl"`
}

// AuthOAuthConfig 是 OAuth 登录配置。
type AuthOAuthConfig struct {
	SuccessRedirectURL string                  `mapstructure:"success_redirect_url"`
	StateTTL           string                  `mapstructure:"state_ttl"`
	LoginCodeTTL       string                  `mapstructure:"login_code_ttl"`
	GitHub             AuthOAuthProviderConfig `mapstructure:"github"`
	Google             AuthOAuthProviderConfig `mapstructure:"google"`
}

// AuthOAuthProviderConfig 是单个 OAuth provider 配置。
type AuthOAuthProviderConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// AuthEmailConfig 是认证邮件配置。
type AuthEmailConfig struct {
	Sender           string `mapstructure:"sender"`
	VerificationTTL  string `mapstructure:"verification_ttl"`
	PasswordResetTTL string `mapstructure:"password_reset_ttl"`
}

// AuthRateLimitConfig 是认证端点限流配置，用于抵御密码爆破/撞库。
type AuthRateLimitConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	RequestsPerMinute int  `mapstructure:"requests_per_minute"`
	Burst             int  `mapstructure:"burst"`
}

// DatabaseConfig 是数据库配置。
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	DSN             string `mapstructure:"dsn"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime int    `mapstructure:"conn_max_idle_time"`
}

// Load 加载应用配置。
func Load() (*Config, error) {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "local"
	}

	dir := os.Getenv("APP_CONFIG_DIR")
	if dir == "" {
		dir = "configs"
	}

	v := viper.New()
	v.SetConfigType("yaml")
	setDefaults(v)
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for _, key := range envKeys() {
		if err := v.BindEnv(key); err != nil {
			return nil, fmt.Errorf("bind env %s: %w", key, err)
		}
	}

	loadedFiles := make([]string, 0, 1)
	envFile := filepath.Join(dir, fmt.Sprintf("config.%s.yaml", env))
	if _, err := os.Stat(envFile); err == nil {
		v.SetConfigFile(envFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config %s: %w", envFile, err)
		}
		loadedFiles = append(loadedFiles, envFile)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat config %s: %w", envFile, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	cfg.Logger.AppEnv = env
	cfg.LoadedFiles = loadedFiles

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate 校验应用配置。
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.Host) == "" {
		return errors.New("server.host 不能为空")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return errors.New("server.port 必须在 1-65535 之间")
	}

	if !oneOf(c.Server.Mode, "debug", "test", "release") {
		return errors.New("server.mode 必须是 debug、test 或 release")
	}

	if !oneOf(c.Logger.Level, "debug", "info", "warn", "error") {
		return errors.New("logger.level 必须是 debug、info、warn 或 error")
	}

	if strings.TrimSpace(c.Logger.Path) == "" {
		return errors.New("logger.path 不能为空")
	}

	if strings.TrimSpace(c.Logger.AppName) == "" {
		return errors.New("logger.app_name 不能为空")
	}

	if c.Audit.MaxBodyBytes < 0 {
		return errors.New("audit.max_body_bytes 必须大于等于 0")
	}

	if !strings.HasPrefix(c.Metrics.Path, "/") {
		return errors.New("metrics.path 必须以 / 开头")
	}

	if !validMetricsNamespace(c.Metrics.Namespace) {
		return errors.New("metrics.namespace 只能包含字母、数字和下划线，且不能以数字开头")
	}

	if c.Sentry.Enabled && strings.TrimSpace(c.Sentry.DSN) == "" {
		return errors.New("sentry.dsn 不能为空")
	}

	if !oneOf(c.Tracing.Sampler, "", "always", "never", "parent", "ratio") {
		return errors.New("tracing.sampler 必须是 always、never、parent 或 ratio")
	}

	if c.Tracing.Sampler == "ratio" {
		if c.Tracing.SampleRatio <= 0 || c.Tracing.SampleRatio > 1 {
			return errors.New("tracing.sample_ratio 必须在 (0, 1] 之间，sampler=ratio 时生效")
		}
	}

	if c.Tracing.Endpoint != "" && c.Tracing.Insecure && isProductionEnv(c.Logger.AppEnv) {
		// 规则要求 prod 环境 tracing.endpoint 非空时 tracing.insecure 必须为 false：
		// 明文 OTLP 容易泄漏链路数据，且与公司安全基线不符，所以直接阻断启动。
		return errors.New("tracing.insecure 在 prod 下必须为 false（明文 OTLP 不被允许）")
	}
	if _, err := time.ParseDuration(c.HTTPClient.DefaultTimeout); err != nil {
		return fmt.Errorf("http_client.default_timeout 无效: %w", err)
	}

	if c.HTTPClient.DefaultResponseBodyLimit < 0 {
		return errors.New("http_client.default_response_body_limit 必须大于等于 0")
	}

	for name, target := range c.HTTPClient.Targets {
		if strings.TrimSpace(name) == "" {
			return errors.New("http_client.targets 不能包含空名称")
		}
		if strings.TrimSpace(target.BaseURL) == "" {
			return fmt.Errorf("http_client.targets.%s.base_url 不能为空", name)
		}
		if target.Timeout != "" {
			if _, err := time.ParseDuration(target.Timeout); err != nil {
				return fmt.Errorf("http_client.targets.%s.timeout 无效: %w", name, err)
			}
		}
		if target.ResponseBodyLimit < 0 {
			return fmt.Errorf("http_client.targets.%s.response_body_limit 必须大于等于 0", name)
		}
	}
	if c.Redis.Enabled {
		if strings.TrimSpace(c.Redis.Addr) == "" {
			return errors.New("redis.addr 不能为空")
		}
		if isReplacementPlaceholder(c.Redis.Addr) || isReplacementPlaceholder(c.Redis.Password) {
			return errors.New("redis 配置不能包含占位符")
		}
	}
	if c.Redis.DB < 0 {
		return errors.New("redis.db 必须大于等于 0")
	}
	if _, err := time.ParseDuration(c.Redis.DialTimeout); err != nil {
		return fmt.Errorf("redis.dial_timeout 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Redis.ReadTimeout); err != nil {
		return fmt.Errorf("redis.read_timeout 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Redis.WriteTimeout); err != nil {
		return fmt.Errorf("redis.write_timeout 无效: %w", err)
	}
	if !oneOf(c.SMTP.TLSMode, "none", "starttls", "tls") {
		return errors.New("smtp.tls_mode 必须是 none、starttls 或 tls")
	}
	if _, err := time.ParseDuration(c.SMTP.Timeout); err != nil {
		return fmt.Errorf("smtp.timeout 无效: %w", err)
	}
	if c.SMTP.Enabled {
		if strings.TrimSpace(c.SMTP.Host) == "" {
			return errors.New("smtp.enabled=true 时 smtp.host 不能为空")
		}
		if c.SMTP.Port <= 0 || c.SMTP.Port > 65535 {
			return errors.New("smtp.port 必须在 1-65535 之间")
		}
		if strings.TrimSpace(c.SMTP.From) == "" {
			return errors.New("smtp.enabled=true 时 smtp.from 不能为空")
		}
		if c.SMTP.Username != "" && strings.TrimSpace(c.SMTP.Password) == "" {
			return errors.New("smtp.username 非空时 smtp.password 不能为空")
		}
		if isReplacementPlaceholder(c.SMTP.Password) {
			return errors.New("smtp.password 不能是占位符")
		}
		if c.Logger.AppEnv == "prod" && c.SMTP.TLSMode == "none" {
			return errors.New("smtp.tls_mode=none 不能用于生产环境")
		}
	}
	if c.Auth.JWT.Enabled {
		if strings.TrimSpace(c.Auth.JWT.Issuer) == "" {
			return errors.New("auth.jwt.issuer 不能为空")
		}
		if strings.TrimSpace(c.Auth.JWT.HMACSecret) == "" {
			return errors.New("auth.jwt.hmac_secret 不能为空")
		}
		if isReplacementPlaceholder(c.Auth.JWT.HMACSecret) {
			return errors.New("auth.jwt.hmac_secret 不能是占位符")
		}
		if isProductionEnv(c.Logger.AppEnv) && len(c.Auth.JWT.HMACSecret) < 32 {
			return errors.New("auth.jwt.hmac_secret 生产环境至少 32 字节")
		}
	}
	if _, err := time.ParseDuration(c.Auth.JWT.ClockSkew); err != nil {
		return fmt.Errorf("auth.jwt.clock_skew 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.JWT.AccessTTL); err != nil {
		return fmt.Errorf("auth.jwt.access_ttl 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.JWT.RefreshTTL); err != nil {
		return fmt.Errorf("auth.jwt.refresh_ttl 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.OAuth.StateTTL); err != nil {
		return fmt.Errorf("auth.oauth.state_ttl 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.OAuth.LoginCodeTTL); err != nil {
		return fmt.Errorf("auth.oauth.login_code_ttl 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.Email.VerificationTTL); err != nil {
		return fmt.Errorf("auth.email.verification_ttl 无效: %w", err)
	}
	if _, err := time.ParseDuration(c.Auth.Email.PasswordResetTTL); err != nil {
		return fmt.Errorf("auth.email.password_reset_ttl 无效: %w", err)
	}
	if c.Auth.RateLimit.Enabled {
		if c.Auth.RateLimit.RequestsPerMinute <= 0 {
			return errors.New("auth.rate_limit.requests_per_minute 启用时必须大于 0")
		}
		if c.Auth.RateLimit.Burst <= 0 {
			return errors.New("auth.rate_limit.burst 启用时必须大于 0")
		}
	}
	if !oneOf(c.Auth.Email.Sender, "log", "disabled") {
		return errors.New("auth.email.sender 必须是 log 或 disabled")
	}
	if isProductionEnv(c.Logger.AppEnv) && c.Auth.Email.Sender == "log" {
		return errors.New("auth.email.sender 生产环境不能是 log")
	}
	if c.Auth.OAuth.GitHub.Enabled {
		if strings.TrimSpace(c.Auth.OAuth.GitHub.ClientID) == "" || strings.TrimSpace(c.Auth.OAuth.GitHub.ClientSecret) == "" || strings.TrimSpace(c.Auth.OAuth.GitHub.RedirectURL) == "" {
			return errors.New("auth.oauth.github 启用时 client_id、client_secret 和 redirect_url 不能为空")
		}
		if isReplacementPlaceholder(c.Auth.OAuth.GitHub.ClientSecret) {
			return errors.New("auth.oauth.github.client_secret 不能是占位符")
		}
	}
	if c.Auth.OAuth.Google.Enabled {
		if strings.TrimSpace(c.Auth.OAuth.Google.ClientID) == "" || strings.TrimSpace(c.Auth.OAuth.Google.ClientSecret) == "" || strings.TrimSpace(c.Auth.OAuth.Google.RedirectURL) == "" {
			return errors.New("auth.oauth.google 启用时 client_id、client_secret 和 redirect_url 不能为空")
		}
		if isReplacementPlaceholder(c.Auth.OAuth.Google.ClientSecret) {
			return errors.New("auth.oauth.google.client_secret 不能是占位符")
		}
	}
	if c.Database.Driver != "mysql" {
		return errors.New("database.driver 必须是 mysql")
	}

	if strings.TrimSpace(c.Database.DSN) == "" {
		return errors.New("database.dsn 不能为空")
	}

	if isReplacementPlaceholder(c.Database.DSN) {
		return errors.New("database.dsn 不能是占位符")
	}

	if c.Database.MaxOpenConns <= 0 {
		return errors.New("database.max_open_conns 必须大于 0")
	}

	if c.Database.MaxIdleConns < 0 {
		return errors.New("database.max_idle_conns 必须大于等于 0")
	}

	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return errors.New("database.max_idle_conns 不能大于 database.max_open_conns")
	}

	if c.Database.ConnMaxLifetime < 0 {
		return errors.New("database.conn_max_lifetime 必须大于等于 0")
	}

	if c.Database.ConnMaxIdleTime < 0 {
		return errors.New("database.conn_max_idle_time 必须大于等于 0")
	}

	return nil
}

func oneOf(value string, allowed ...string) bool {
	return slices.Contains(allowed, value)
}

func validMetricsNamespace(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if isASCIILetter(r) || r == '_' || (index > 0 && isASCIIDigit(r)) {
			continue
		}
		return false
	}
	return true
}
func isReplacementPlaceholder(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "REPLACE_")
}

// isProductionEnv 判定运行环境是否按生产级处理。
// 采用 fail-safe 策略：只有显式声明的非生产环境（local/dev/test）才放宽校验，
// 其余取值（prod、production、prd、staging 或任何未知值）一律按生产严格校验，
// 避免 APP_ENV=production 这类常见别名静默绕过生产安全护栏。
func isProductionEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "local", "dev", "development", "test", "testing", "example":
		return false
	default:
		return true
	}
}

func isASCIILetter(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "release")
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.path", "./logs")
	v.SetDefault("logger.app_name", "gotobeta")
	v.SetDefault("audit.enabled", true)
	v.SetDefault("audit.log_request_body", true)
	v.SetDefault("audit.log_response_body", false)
	v.SetDefault("audit.mask_sensitive_fields", true)
	v.SetDefault("audit.max_body_bytes", 65536)
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.namespace", "gotobeta")
	v.SetDefault("sentry.enabled", false)
	v.SetDefault("sentry.dsn", "")
	v.SetDefault("sentry.env", "local")
	v.SetDefault("tracing.endpoint", "")
	v.SetDefault("tracing.insecure", false)
	v.SetDefault("tracing.sampler", "parent")
	v.SetDefault("tracing.sample_ratio", 0.1)
	v.SetDefault("tracing.service_name", "gotobeta")
	v.SetDefault("tracing.service_version", "")
	v.SetDefault("smtp.enabled", false)
	v.SetDefault("smtp.host", "127.0.0.1")
	v.SetDefault("smtp.port", 1025)
	v.SetDefault("smtp.username", "")
	v.SetDefault("smtp.password", "")
	v.SetDefault("smtp.from", "gotobeta <no-reply@example.com>")
	v.SetDefault("smtp.tls_mode", "none")
	v.SetDefault("smtp.timeout", "5s")
	v.SetDefault("http_client.default_timeout", "5s")
	v.SetDefault("http_client.default_response_body_limit", 1048576)
	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("redis.key_prefix", "gotobeta:")
	v.SetDefault("auth.jwt.enabled", true)
	v.SetDefault("auth.jwt.issuer", "gotobeta")
	v.SetDefault("auth.jwt.audience", "")
	// HMAC 密钥默认留空，强制 fail-closed：与 database.dsn 一致，未显式提供即被
	// Validate 拒绝。开发态密钥由 configs/config.{local,dev,test}.yaml 注入，
	// 不在代码默认值里硬编码可过校验的弱密钥，杜绝生产别名环境静默使用默认密钥。
	v.SetDefault("auth.jwt.hmac_secret", "")
	v.SetDefault("auth.jwt.clock_skew", "30s")
	v.SetDefault("auth.jwt.access_ttl", "15m")
	v.SetDefault("auth.jwt.refresh_ttl", "720h")
	v.SetDefault("auth.oauth.success_redirect_url", "http://localhost:3000/auth/callback")
	v.SetDefault("auth.oauth.state_ttl", "10m")
	v.SetDefault("auth.oauth.login_code_ttl", "2m")
	v.SetDefault("auth.oauth.github.enabled", false)
	v.SetDefault("auth.oauth.github.client_id", "")
	v.SetDefault("auth.oauth.github.client_secret", "")
	v.SetDefault("auth.oauth.github.redirect_url", "http://localhost:8080/api/v1/auth/oauth/github/callback")
	v.SetDefault("auth.oauth.google.enabled", false)
	v.SetDefault("auth.oauth.google.client_id", "")
	v.SetDefault("auth.oauth.google.client_secret", "")
	v.SetDefault("auth.oauth.google.redirect_url", "http://localhost:8080/api/v1/auth/oauth/google/callback")
	v.SetDefault("auth.email.sender", "log")
	v.SetDefault("auth.email.verification_ttl", "24h")
	v.SetDefault("auth.email.password_reset_ttl", "1h")
	v.SetDefault("auth.rate_limit.enabled", true)
	v.SetDefault("auth.rate_limit.requests_per_minute", 60)
	v.SetDefault("auth.rate_limit.burst", 10)
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 300)
	v.SetDefault("database.conn_max_idle_time", 180)
	v.SetDefault("im.recall_window", "2m")
	v.SetDefault("im.presence_ttl", "30s")
	v.SetDefault("im.ws_ticket_ttl", "30s")
	v.SetDefault("im.message_page_size", 100)
	v.SetDefault("im.message_rate_per_minute", 120)
	v.SetDefault("im.message_rate_burst", 20)
	v.SetDefault("objstore.endpoint", "")
	v.SetDefault("objstore.region", "")
	v.SetDefault("objstore.bucket", "")
	v.SetDefault("objstore.access_key", "")
	v.SetDefault("objstore.secret_key", "")
	v.SetDefault("objstore.use_ssl", false)
	v.SetDefault("objstore.public_base_url", "")
	v.SetDefault("objstore.presign_ttl", "15m")
}

func envKeys() []string {
	keys := []string{
		"server.host",
		"server.port",
		"server.mode",
		"logger.level",
		"logger.path",
		"logger.app_name",
		"audit.enabled",
		"audit.log_request_body",
		"audit.log_response_body",
		"audit.mask_sensitive_fields",
		"audit.max_body_bytes",
		"metrics.enabled",
		"metrics.path",
		"metrics.namespace",
		"sentry.enabled",
		"sentry.dsn",
		"sentry.env",
		"tracing.endpoint",
		"tracing.insecure",
		"tracing.sampler",
		"tracing.sample_ratio",
		"tracing.service_name",
		"tracing.service_version",
		"smtp.enabled",
		"smtp.host",
		"smtp.port",
		"smtp.username",
		"smtp.password",
		"smtp.from",
		"smtp.tls_mode",
		"smtp.timeout",
		"http_client.default_timeout",
		"http_client.default_response_body_limit",
		"redis.enabled",
		"redis.addr",
		"redis.password",
		"redis.db",
		"redis.dial_timeout",
		"redis.read_timeout",
		"redis.write_timeout",
		"redis.key_prefix",
		"auth.jwt.enabled",
		"auth.jwt.issuer",
		"auth.jwt.audience",
		"auth.jwt.hmac_secret",
		"auth.jwt.clock_skew",
		"auth.jwt.access_ttl",
		"auth.jwt.refresh_ttl",
		"auth.oauth.success_redirect_url",
		"auth.oauth.state_ttl",
		"auth.oauth.login_code_ttl",
		"auth.oauth.github.enabled",
		"auth.oauth.github.client_id",
		"auth.oauth.github.client_secret",
		"auth.oauth.github.redirect_url",
		"auth.oauth.google.enabled",
		"auth.oauth.google.client_id",
		"auth.oauth.google.client_secret",
		"auth.oauth.google.redirect_url",
		"auth.email.sender",
		"auth.email.verification_ttl",
		"auth.email.password_reset_ttl",
		"auth.rate_limit.enabled",
		"auth.rate_limit.requests_per_minute",
		"auth.rate_limit.burst",
	}
	keys = append(keys,
		"database.driver",
		"database.dsn",
		"database.max_open_conns",
		"database.max_idle_conns",
		"database.conn_max_lifetime",
		"database.conn_max_idle_time",
	)
	keys = append(keys,
		"im.recall_window",
		"im.presence_ttl",
		"im.ws_ticket_ttl",
		"im.message_page_size",
		"im.message_rate_per_minute",
		"im.message_rate_burst",
		"objstore.endpoint",
		"objstore.region",
		"objstore.bucket",
		"objstore.access_key",
		"objstore.secret_key",
		"objstore.use_ssl",
		"objstore.public_base_url",
		"objstore.presign_ttl",
	)

	return keys
}
