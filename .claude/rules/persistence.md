---
paths:
  - "internal/modules/**/infra/persistence/**"
  - "internal/infra/entdb/**"
  - "internal/pkg/persistence/**"
---

# Persistence

- persistence 层负责把领域仓储接口适配到 Ent/MySQL 或其他存储实现。
- 不要把 Ent 实体直接泄漏到 domain 或 adapter 层。
- 事务边界由 application 或 bootstrap 明确传入，不在深层函数里隐式开启全局事务。
- 查询条件必须由已校验输入构造，禁止拼接 SQL 字符串。
- 数据库表之间禁止使用外键；通过业务唯一键、索引和应用层一致性维护关系。
- `make test-architecture` 会自动校验模块 infra 层不得反向 import application、adapter 或 Gin。
