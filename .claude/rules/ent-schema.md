---
paths:
  - "internal/ent/schema/**"
  - "internal/app/server/migrate.go"
---

# Ent Schema

- schema 只描述数据库结构、索引和字段约束，不放业务流程逻辑。
- 所有表必须有稳定业务 ID 字段，避免把自增 ID 暴露给外部接口。
- 时间字段使用统一 mixin，避免各 schema 重复定义。
- 不定义数据库外键；需要关系时使用普通字段和索引表达。
- 修改 schema 后运行 `make generate`，并同步迁移入口和测试。
