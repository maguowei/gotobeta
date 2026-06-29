# 领域事件文档

记录本服务对外发布和订阅的领域事件。事件文档是契约，不只是实现备注；新增事件前先补表格，再实现代码和测试。

当前 preset 未启用异步事件模块。保留本文档是为了在后续引入 `event-core`、`event-bus-kafka` 或 `event-inbox` 时先定义契约；在模块启用前，不要把下方 outbox/inbox 约束当成已存在的运行能力。

## 实现约束

- 业务事务内只写 outbox，不在 application service 中直接调用 Kafka。
- topic 来自配置，默认示例是 `gotobeta.events`；真实环境通过配置覆盖，禁止在代码或文档中写死真实 topic。
- outbox/inbox 时间字段使用 MySQL `datetime(3)`，避免亚秒精度丢失导致立即认领失败。
- worker 认领使用租约持有者、租约过期时间、下次重试时间和 `FOR UPDATE SKIP LOCKED`。
- 同一 `partition_key` 或 `aggregate_key` 的事件在同一批次只认领最早一条，保持同聚合顺序。
- 每条事件独立 recover，panic 会进入 retry/dead 流程，不影响同批其他事件。
- inbox 幂等键必须来自业务稳定事实，不使用本地自增 ID；`event_id` 和 `idempotency_key` 的唯一性按 `consumer_name` 作用域约束，允许多个消费者复用同一个上游事件。
- 暂不消费的事件类型需要通过 registrar 显式忽略，并通过 eventbus metrics 留痕。
- Prometheus label 只使用有限集合：component、event_type、status，禁止放入动态 ID 或错误文本。

## 发布事件

| 事件类型 | 触发场景 | Partition Key | Payload 说明 | Schema/版本 |
|---|---|---|---|---|
| `example.event.created` | 示例触发场景 | `aggregate_id` | JSON payload | `v1` |

## 订阅事件

| 事件类型 | 来源服务 | Consumer Name | Idempotency Key | 处理逻辑 |
|---|---|---|---|---|
| `example.event.created` | `source-service` | `gotobeta` | 业务稳定键 | 幂等处理 |

## 变更清单

1. 先在本文件登记事件类型、payload、partition/idempotency key 和版本。
2. 发布事件时，在 application 事务内调用 outbox writer，确保业务写入和事件写入同事务。
3. 消费事件时，为每个 handler 明确 `consumer_name`，并为未知或暂不消费事件配置显式忽略。
4. 新增或调整重试策略时，同步 eventbus metrics、日志字段和 release note。
5. 完成前运行 `make test`、`make test-architecture`；涉及 worker、Docker 或 K8s 时补 `make smoke`、`docker compose config` 或 K8s dry-run。
