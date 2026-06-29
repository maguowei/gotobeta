---
paths:
  - "api/openapi.yaml"
  - "internal/modules/**/adapter/http/**"
---

# OpenAPI Contract

- 新增或修改 HTTP API 时，先更新 `api/openapi.yaml`，再实现 handler 和 DTO。
- OpenAPI 中的 request/response schema 必须与代码中的 DTO 字段保持一致。
- 错误响应统一引用公共错误结构，避免每个接口自定义不兼容格式。
- 路径命名保持资源化和版本化，默认放在 `/api/v1` 下。
- 删除或改名接口时同步更新 README、测试和调用方示例。
