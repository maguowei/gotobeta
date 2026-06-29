package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPresetConfigs(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "configs")
	envs := []string{"example", "local", "dev", "test", "prod"}

	for _, env := range envs {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			t.Setenv("APP_CONFIG_DIR", configDir)
			if env == "prod" {
				t.Setenv("APP_DATABASE_DSN", "root:password@tcp(mysql:3306)/gotobeta?parseTime=true&charset=utf8mb4")
			}
			if env == "prod" {
				t.Setenv("APP_AUTH_JWT_HMAC_SECRET", "prod-test-hmac-secret-at-least-32-bytes")
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if len(cfg.LoadedFiles) != 1 {
				t.Fatalf("LoadedFiles length = %d, want 1", len(cfg.LoadedFiles))
			}
		})
	}
}
func TestLoadAppliesIMAndObjStoreDefaults(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "configs")
	t.Setenv("APP_ENV", "example")
	t.Setenv("APP_CONFIG_DIR", configDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.IM.RecallWindow != "2m" {
		t.Fatalf("IM.RecallWindow = %q, want 2m", cfg.IM.RecallWindow)
	}
	if cfg.IM.MessagePageSize != 100 {
		t.Fatalf("IM.MessagePageSize = %d, want 100", cfg.IM.MessagePageSize)
	}
	if cfg.IM.MessageRatePerMinute != 120 {
		t.Fatalf("IM.MessageRatePerMinute = %d, want 120", cfg.IM.MessageRatePerMinute)
	}
	if cfg.IM.MessageRateBurst != 20 {
		t.Fatalf("IM.MessageRateBurst = %d, want 20", cfg.IM.MessageRateBurst)
	}
	if cfg.IM.MaxWSConnections != 10000 {
		t.Fatalf("IM.MaxWSConnections = %d, want 10000", cfg.IM.MaxWSConnections)
	}
	if cfg.IM.MaxConnPerUser != 10 {
		t.Fatalf("IM.MaxConnPerUser = %d, want 10", cfg.IM.MaxConnPerUser)
	}
	if cfg.IM.WSHandshakeRatePerMinute != 60 {
		t.Fatalf("IM.WSHandshakeRatePerMinute = %d, want 60", cfg.IM.WSHandshakeRatePerMinute)
	}
	if cfg.IM.WSTicketTTL != "30s" {
		t.Fatalf("IM.WSTicketTTL = %q, want 30s", cfg.IM.WSTicketTTL)
	}
	if cfg.ObjStore.PresignTTL != "15m" {
		t.Fatalf("ObjStore.PresignTTL = %q, want 15m", cfg.ObjStore.PresignTTL)
	}
}

func TestLoadRejectsProductionPlaceholdersWithoutRuntimeOverrides(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "configs")
	t.Setenv("APP_ENV", "prod")
	t.Setenv("APP_CONFIG_DIR", configDir)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want production placeholder rejection")
	}
	if !strings.Contains(err.Error(), "占位符") {
		t.Fatalf("Load() error = %q, want placeholder rejection", err.Error())
	}
}

func TestLoadAllowsContainerRuntimeEnvOverrides(t *testing.T) {
	configDir := filepath.Join("..", "..", "..", "configs")
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_CONFIG_DIR", configDir)
	t.Setenv("APP_DATABASE_DSN", "root:password@tcp(mysql:3306)/gotobeta?parseTime=true&charset=utf8mb4")
	t.Setenv("APP_AUTH_JWT_HMAC_SECRET", "prod-test-hmac-secret-at-least-32-bytes")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Database.DSN != "root:password@tcp(mysql:3306)/gotobeta?parseTime=true&charset=utf8mb4" {
		t.Fatalf("Database.DSN = %q, want container DSN", cfg.Database.DSN)
	}
	if cfg.Auth.JWT.HMACSecret != "prod-test-hmac-secret-at-least-32-bytes" {
		t.Fatalf("Auth.JWT.HMACSecret = %q, want runtime secret", cfg.Auth.JWT.HMACSecret)
	}
}
func TestLoadRejectsProductionLogEmailSenderFallback(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("APP_CONFIG_DIR", t.TempDir())
	t.Setenv("APP_DATABASE_DSN", "root:password@tcp(mysql:3306)/gotobeta?parseTime=true&charset=utf8mb4")
	t.Setenv("APP_AUTH_JWT_HMAC_SECRET", "prod-test-hmac-secret-at-least-32-bytes")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want production log sender rejection")
	}
	if !strings.Contains(err.Error(), "auth.email.sender 生产环境不能是 log") {
		t.Fatalf("Load() error = %q, want log sender rejection", err.Error())
	}
}

func TestLoadTreatsProductionAliasAsProduction(t *testing.T) {
	// 回归保护 P0：APP_ENV=production 等别名必须按生产级严格校验。
	// 否则当 config.production.yaml 缺失而静默回退默认值时，过短/默认 HMAC
	// 密钥会被接受，攻击者可用公开已知密钥伪造任意用户的 JWT 绕过认证。
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_CONFIG_DIR", t.TempDir())
	t.Setenv("APP_DATABASE_DSN", "root:password@tcp(mysql:3306)/gotobeta?parseTime=true&charset=utf8mb4")
	// 故意提供一个不足 32 字节的密钥：生产级校验必须拒绝它。
	t.Setenv("APP_AUTH_JWT_HMAC_SECRET", "too-short-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want production-strict rejection for APP_ENV=production")
	}
	if !strings.Contains(err.Error(), "auth.jwt.hmac_secret 生产环境至少 32 字节") {
		t.Fatalf("Load() error = %q, want hmac length rejection under production alias", err.Error())
	}
}

func TestLoadFailsClosedWhenHMACSecretMissing(t *testing.T) {
	// 回归保护 P0：HMAC 密钥默认必须 fail-closed（空），未显式提供即拒绝启动，
	// 不得依赖代码里硬编码的可过校验默认密钥。
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_CONFIG_DIR", t.TempDir())
	t.Setenv("APP_DATABASE_DSN", "root:password@tcp(mysql:3306)/gotobeta?parseTime=true&charset=utf8mb4")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want rejection when hmac_secret is unset")
	}
	if !strings.Contains(err.Error(), "auth.jwt.hmac_secret 不能为空") {
		t.Fatalf("Load() error = %q, want empty hmac rejection", err.Error())
	}
}

func TestValidateRejectsInvalidRuntimeSettings(t *testing.T) {
	testCases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "server host empty",
			mutate: func(cfg *Config) {
				cfg.Server.Host = " "
			},
			wantErr: "server.host 不能为空",
		},
		{
			name: "server port too low",
			mutate: func(cfg *Config) {
				cfg.Server.Port = 0
			},
			wantErr: "server.port 必须在 1-65535 之间",
		},
		{
			name: "server port too high",
			mutate: func(cfg *Config) {
				cfg.Server.Port = 70000
			},
			wantErr: "server.port 必须在 1-65535 之间",
		},
		{
			name: "server mode unsupported",
			mutate: func(cfg *Config) {
				cfg.Server.Mode = "verbose"
			},
			wantErr: "server.mode 必须是 debug、test 或 release",
		},
		{
			name: "logger level unsupported",
			mutate: func(cfg *Config) {
				cfg.Logger.Level = "trace"
			},
			wantErr: "logger.level 必须是 debug、info、warn 或 error",
		},
		{
			name: "logger path empty",
			mutate: func(cfg *Config) {
				cfg.Logger.Path = " "
			},
			wantErr: "logger.path 不能为空",
		},
		{
			name: "logger app name empty",
			mutate: func(cfg *Config) {
				cfg.Logger.AppName = " "
			},
			wantErr: "logger.app_name 不能为空",
		},
		{
			name: "audit max body bytes negative",
			mutate: func(cfg *Config) {
				cfg.Audit.MaxBodyBytes = -1
			},
			wantErr: "audit.max_body_bytes 必须大于等于 0",
		},
		{
			name: "metrics path is relative",
			mutate: func(cfg *Config) {
				cfg.Metrics.Path = "metrics"
			},
			wantErr: "metrics.path 必须以 / 开头",
		},
		{
			name: "metrics namespace invalid",
			mutate: func(cfg *Config) {
				cfg.Metrics.Namespace = "bad-name"
			},
			wantErr: "metrics.namespace 只能包含字母、数字和下划线，且不能以数字开头",
		},
		{
			name: "sentry enabled without dsn",
			mutate: func(cfg *Config) {
				cfg.Sentry.Enabled = true
				cfg.Sentry.DSN = ""
			},
			wantErr: "sentry.dsn 不能为空",
		},
		{
			name: "tracing sampler unsupported",
			mutate: func(cfg *Config) {
				cfg.Tracing.Sampler = "random"
			},
			wantErr: "tracing.sampler 必须是 always、never、parent 或 ratio",
		},
		{
			name: "tracing ratio too low",
			mutate: func(cfg *Config) {
				cfg.Tracing.Sampler = "ratio"
				cfg.Tracing.SampleRatio = 0
			},
			wantErr: "tracing.sample_ratio 必须在 (0, 1] 之间",
		},
		{
			name: "tracing ratio too high",
			mutate: func(cfg *Config) {
				cfg.Tracing.Sampler = "ratio"
				cfg.Tracing.SampleRatio = 1.1
			},
			wantErr: "tracing.sample_ratio 必须在 (0, 1] 之间",
		},
		{
			name: "prod tracing endpoint with insecure must be rejected",
			mutate: func(cfg *Config) {
				cfg.Logger.AppEnv = "prod"
				cfg.Tracing.Endpoint = "otel-collector:4317"
				cfg.Tracing.Insecure = true
			},
			wantErr: "tracing.insecure 在 prod 下必须为 false",
		},
		{
			name: "smtp tls mode unsupported",
			mutate: func(cfg *Config) {
				cfg.SMTP.TLSMode = "opportunistic"
			},
			wantErr: "smtp.tls_mode 必须是 none、starttls 或 tls",
		},
		{
			name: "smtp enabled without host",
			mutate: func(cfg *Config) {
				cfg.SMTP.Enabled = true
				cfg.SMTP.Host = " "
			},
			wantErr: "smtp.enabled=true 时 smtp.host 不能为空",
		},
		{
			name: "smtp enabled with invalid port",
			mutate: func(cfg *Config) {
				cfg.SMTP.Enabled = true
				cfg.SMTP.Port = 0
			},
			wantErr: "smtp.port 必须在 1-65535 之间",
		},
		{
			name: "prod smtp plaintext rejected",
			mutate: func(cfg *Config) {
				cfg.Logger.AppEnv = "prod"
				cfg.SMTP.Enabled = true
				cfg.SMTP.TLSMode = "none"
			},
			wantErr: "smtp.tls_mode=none 不能用于生产环境",
		},
		{
			name: "http client default timeout invalid",
			mutate: func(cfg *Config) {
				cfg.HTTPClient.DefaultTimeout = "soon"
			},
			wantErr: "http_client.default_timeout 无效",
		},
		{
			name: "http client default body limit negative",
			mutate: func(cfg *Config) {
				cfg.HTTPClient.DefaultResponseBodyLimit = -1
			},
			wantErr: "http_client.default_response_body_limit 必须大于等于 0",
		},
		{
			name: "http client target base url empty",
			mutate: func(cfg *Config) {
				cfg.HTTPClient.Targets = map[string]HTTPClientTarget{"billing": {BaseURL: " "}}
			},
			wantErr: "http_client.targets.billing.base_url 不能为空",
		},
		{
			name: "redis enabled with empty addr",
			mutate: func(cfg *Config) {
				cfg.Redis.Enabled = true
				cfg.Redis.Addr = " "
			},
			wantErr: "redis.addr 不能为空",
		},
		{
			name: "redis db negative",
			mutate: func(cfg *Config) {
				cfg.Redis.DB = -1
			},
			wantErr: "redis.db 必须大于等于 0",
		},
		{
			name: "redis read timeout invalid",
			mutate: func(cfg *Config) {
				cfg.Redis.ReadTimeout = "later"
			},
			wantErr: "redis.read_timeout 无效",
		},
		{
			name: "auth jwt enabled without issuer",
			mutate: func(cfg *Config) {
				cfg.Auth.JWT.Enabled = true
				cfg.Auth.JWT.Issuer = " "
			},
			wantErr: "auth.jwt.issuer 不能为空",
		},
		{
			name: "auth jwt enabled without hmac secret",
			mutate: func(cfg *Config) {
				cfg.Auth.JWT.Enabled = true
				cfg.Auth.JWT.HMACSecret = ""
			},
			wantErr: "auth.jwt.hmac_secret 不能为空",
		},
		{
			name: "prod auth jwt hmac secret too short",
			mutate: func(cfg *Config) {
				cfg.Logger.AppEnv = "prod"
				cfg.Auth.JWT.Enabled = true
				cfg.Auth.JWT.HMACSecret = "short"
			},
			wantErr: "auth.jwt.hmac_secret 生产环境至少 32 字节",
		},
		{
			name: "auth jwt clock skew invalid",
			mutate: func(cfg *Config) {
				cfg.Auth.JWT.ClockSkew = "loose"
			},
			wantErr: "auth.jwt.clock_skew 无效",
		},
		{
			name: "prod email log sender rejected",
			mutate: func(cfg *Config) {
				cfg.Logger.AppEnv = "prod"
				cfg.Auth.JWT.HMACSecret = "prod-test-hmac-secret-at-least-32-bytes"
				cfg.Auth.Email.Sender = "log"
			},
			wantErr: "auth.email.sender 生产环境不能是 log",
		},
		{
			name: "database dsn empty",
			mutate: func(cfg *Config) {
				cfg.Database.DSN = " "
			},
			wantErr: "database.dsn 不能为空",
		},
		{
			name: "database dsn is replacement placeholder",
			mutate: func(cfg *Config) {
				cfg.Database.DSN = "REPLACE_VIA_APP_DATABASE_DSN"
			},
			wantErr: "database.dsn 不能是占位符",
		},
		{
			name: "database driver unsupported",
			mutate: func(cfg *Config) {
				cfg.Database.Driver = "unsupported"
			},
			wantErr: "database.driver 必须是 mysql",
		},
		{
			name: "database max open conns invalid",
			mutate: func(cfg *Config) {
				cfg.Database.MaxOpenConns = 0
			},
			wantErr: "database.max_open_conns 必须大于 0",
		},
		{
			name: "database max idle conns negative",
			mutate: func(cfg *Config) {
				cfg.Database.MaxIdleConns = -1
			},
			wantErr: "database.max_idle_conns 必须大于等于 0",
		},
		{
			name: "database max idle conns too high",
			mutate: func(cfg *Config) {
				cfg.Database.MaxOpenConns = 2
				cfg.Database.MaxIdleConns = 3
			},
			wantErr: "database.max_idle_conns 不能大于 database.max_open_conns",
		},
		{
			name: "database conn max lifetime negative",
			mutate: func(cfg *Config) {
				cfg.Database.ConnMaxLifetime = -1
			},
			wantErr: "database.conn_max_lifetime 必须大于等于 0",
		},
		{
			name: "database conn max idle time negative",
			mutate: func(cfg *Config) {
				cfg.Database.ConnMaxIdleTime = -1
			},
			wantErr: "database.conn_max_idle_time 必须大于等于 0",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := validConfig()
			testCase.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() error = nil, want %q", testCase.wantErr)
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("Validate() error = %q, want contains %q", err.Error(), testCase.wantErr)
			}
		})
	}
}

func validConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "release",
		},
		Logger: LoggerConfig{
			Level:   "info",
			Path:    "./logs",
			AppName: "gotobeta",
			AppEnv:  "test",
		},
		Audit: AuditConfig{
			Enabled:             true,
			LogRequestBody:      true,
			LogResponseBody:     false,
			MaskSensitiveFields: true,
			MaxBodyBytes:        65536,
		},
		Metrics: MetricsConfig{
			Enabled:   true,
			Path:      "/metrics",
			Namespace: "gotobeta",
		},
		Sentry: SentryConfig{
			Enabled: false,
			DSN:     "",
			Env:     "test",
		},
		SMTP: SMTPConfig{
			Enabled: false,
			Host:    "127.0.0.1",
			Port:    1025,
			From:    "gotobeta <no-reply@example.com>",
			TLSMode: "none",
			Timeout: "5s",
		},
		HTTPClient: HTTPClientConfig{
			DefaultTimeout:           "5s",
			DefaultResponseBodyLimit: 1048576,
			Targets: map[string]HTTPClientTarget{
				"example": {
					BaseURL:           "https://example.com",
					Timeout:           "3s",
					ResponseBodyLimit: 524288,
				},
			},
		},
		Redis: RedisConfig{
			Enabled:      false,
			Addr:         "127.0.0.1:6379",
			Password:     "",
			DB:           0,
			DialTimeout:  "5s",
			ReadTimeout:  "3s",
			WriteTimeout: "3s",
			KeyPrefix:    "gotobeta:",
		},
		Auth: AuthConfig{
			JWT: AuthJWTConfig{
				Enabled:    true,
				Issuer:     "gotobeta",
				Audience:   "",
				HMACSecret: "local-dev-hmac-secret-change-me",
				ClockSkew:  "30s",
				AccessTTL:  "15m",
				RefreshTTL: "720h",
			},
			OAuth: AuthOAuthConfig{
				SuccessRedirectURL: "http://localhost:3000/auth/callback",
				StateTTL:           "10m",
				LoginCodeTTL:       "2m",
				GitHub: AuthOAuthProviderConfig{
					Enabled:     false,
					RedirectURL: "http://localhost:8080/api/v1/auth/oauth/github/callback",
				},
				Google: AuthOAuthProviderConfig{
					Enabled:     false,
					RedirectURL: "http://localhost:8080/api/v1/auth/oauth/google/callback",
				},
			},
			Email: AuthEmailConfig{
				Sender:           "log",
				VerificationTTL:  "24h",
				PasswordResetTTL: "1h",
			},
		},
		Database: DatabaseConfig{
			Driver:          "mysql",
			DSN:             "user:pass@tcp(127.0.0.1:3306)/gotobeta",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 300,
			ConnMaxIdleTime: 180,
		},
	}
}
