# 配置说明

配置优先级：

```text
APP_* 环境变量 > config.{APP_ENV}.yaml > 默认值
```

默认 `APP_ENV=local`，默认配置目录为 `configs/`。

## 文件说明

| 文件 | 用途 |
|---|---|
| `config.example.yaml` | 字段样例和默认参考 |
| `config.local.yaml` | 本地开发 |
| `config.dev.yaml` | 开发/测试环境部署 |
| `config.test.yaml` | 自动化测试 |
| `config.prod.yaml` | 生产部署模板 |

## 环境变量

配置键会绑定到 `APP_` 前缀环境变量，例如：

```bash
APP_SERVER_PORT=18080
APP_LOGGER_LEVEL=debug
APP_SENTRY_DSN=https://example@sentry.local/1
```
数据库 DSN 可通过 `APP_DATABASE_DSN` 注入，生产配置不要提交真实账号密码。
Redis 可通过 `APP_REDIS_ENABLED` 和 `APP_REDIS_ADDR` 启用并指定地址；生产密码通过环境变量或 Secret 注入。
JWT HMAC secret 必须通过 `APP_AUTH_JWT_HMAC_SECRET` 或 Secret 注入，生产环境至少 32 字节，且不要提交真实密钥。

## 校验约束

- `server.port` 必须在 `1-65535` 之间，`server.mode` 只能是 `debug`、`test` 或 `release`。
- `logger.level` 只能是 `debug`、`info`、`warn` 或 `error`。
- `metrics.path` 必须以 `/` 开头，`metrics.namespace` 只能包含字母、数字和下划线，且不能以数字开头。
- `sentry.enabled=true` 时必须通过配置文件或 `APP_SENTRY_DSN` 提供 DSN。
- `tracing.sampler` 必须是 `always`、`never`、`parent` 或 `ratio`；为 `ratio` 时 `tracing.sample_ratio` 必须在 (0, 1]。
- `database.driver` 固定为 `mysql`，`database.dsn` 不能为空且不能保留 `REPLACE_` 占位符；连接池参数必须非负，且 `max_idle_conns` 不能大于 `max_open_conns`。
- `http_client.default_timeout` 和每个 target 的 `timeout` 必须是合法 duration；response body limit 必须非负；target `base_url` 不能为空。
- `redis.enabled=true` 时 `redis.addr` 不能为空且不能保留 `REPLACE_` 占位符；DB 不能为负数；timeout 必须是合法 duration。
- `smtp.tls_mode` 只能是 `none`、`starttls` 或 `tls`；`smtp.enabled=true` 时 host、port、from 必须有效，生产环境禁止 `tls_mode=none`。
- `auth.jwt.enabled=true` 时 `issuer` 和 `hmac_secret` 不能为空，且 `hmac_secret` 不能保留 `REPLACE_` 占位符；生产环境 `hmac_secret` 至少 32 字节；`clock_skew` 必须是合法 duration。
- `auth.oauth.success_redirect_url` 是 OAuth 登录码唯一允许的前端回跳地址；请求中的 `redirect_url` 为空时使用它，非空时必须完全匹配。
- `auth.email.sender=log` 仅允许本地开发和测试使用；生产环境必须配置为 `disabled` 或接入真实发送器，不能把一次性 token 写入日志。
- `auth.rate_limit.enabled=true`（默认）对登录、注册、刷新、密码找回/重置、邮箱验证、OAuth 码兑换等凭据敏感端点按客户端 IP 限流，抵御在线密码爆破/撞库；`requests_per_minute` 为稳态速率、`burst` 为突发容量，启用时两者必须大于 0。

## Tracing 配置

| 字段 | 默认 | 说明 |
|---|---|---|
| `tracing.endpoint` | `""` | OTLP/gRPC collector 地址；非空即启用，为空时使用 noop provider（零开销） |
| `tracing.insecure` | `true` (prod 为 `false`) | 内网常用；生产推荐 `false` 走 TLS |
| `tracing.sampler` | `parent` | `always` / `never` / `parent` / `ratio` |
| `tracing.sample_ratio` | `0.1` | 仅 `sampler=ratio` 时生效，取值 (0, 1] |
| `tracing.service_name` | `gotobeta` | 上报到 collector 的服务名 |
| `tracing.service_version` | `""` | 服务版本，便于按版本维度筛查 |

环境变量映射：`APP_TRACING_ENDPOINT`、`APP_TRACING_SAMPLER`、`APP_TRACING_SAMPLE_RATIO` 等。短命进程（`migrate`、`datainit`）的 `bootstrap.Options.EnableTracer=false` 会强制 noop，与配置无关。
