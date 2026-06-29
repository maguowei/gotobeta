---
paths:
  - "internal/infra/eventbus/**"
  - "internal/pkg/event/**"
  - "cmd/worker/**"
---

# 事件总线规范

## 基本原则

- 领域事件接口定义在 `internal/pkg/event/`，不依赖任何基础设施
- 具体事件（如 `todo.CreatedEvent`）放在各模块对应聚合包内（如 `domain/todo/events.go`）
- 事件发布机制（Outbox、Kafka、内存总线）放在 `internal/infra/eventbus/`
- 业务操作与事件写入必须在同一事务内完成
- 业务事务内只写 outbox，发布 Kafka 由独立 worker 负责

## Outbox/Inbox

- outbox/inbox 时间字段使用 MySQL `datetime(3)`。
- worker claim 必须使用 lease owner、lease duration、next retry 和 `FOR UPDATE SKIP LOCKED`。
- 同一 `partition_key` 或 `aggregate_key` 的事件必须保持顺序，批量认领时只取最早一条。
- 单条事件独立 recover，panic 转换为 retry/dead，不允许让整批消息失去状态更新。
- inbox 幂等键来自业务稳定事实，不使用自增 ID；唯一性按 `consumer_name` 作用域约束，不做全局唯一。
- 暂不消费的事件类型必须显式登记忽略，并通过指标记录。

## 可观测性

- outbox/inbox 必须调用 `metrics.ObserveEventBus`。
- Prometheus label 禁止使用动态 ID、原始错误文本、完整 URL。
- status label 使用有限集合，例如 `published`、`processed`、`ignored`、`retry`、`dead`、`error`。
