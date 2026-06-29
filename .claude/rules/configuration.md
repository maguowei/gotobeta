---
paths:
  - "configs/**"
  - "internal/infra/config/**"
---

# Configuration

- 配置结构、默认值、环境变量绑定和校验必须同步修改。
- `APP_ENV` 决定加载 `configs/config.<env>.yaml`，`APP_CONFIG_DIR` 可覆盖配置目录。
- 环境变量统一使用 `APP_` 前缀和下划线形式，例如 `APP_SERVER_PORT`。
- 密钥、DSN、Token 不提交真实值；生产配置使用环境变量注入。
- 新增配置项必须补充 `configs/README.md` 并覆盖至少一个测试或启动验证场景。
- `Validate()` 必须拒绝非法运行时边界：端口范围、Gin mode、日志级别、metrics 路径和 namespace、启用 Sentry 时缺失 DSN、数据库 driver/pool 配置。
