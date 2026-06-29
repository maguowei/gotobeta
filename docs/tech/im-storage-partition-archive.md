# IM 消息存储：分区、归档与逻辑外键

适用范围：`internal/ent/schema/message.go`、`conversation.go` 及其生成代码。
本文档记录阶段 A 的存储设计取舍，并给出阶段 B 宽列/分区演进的迁移路径。

## 1. 逻辑外键（无数据库外键）

项目约定**不建数据库外键**，跨聚合一致性由应用层和唯一索引保证（见 `CLAUDE.md`）。
为保留关系语义，跨聚合引用字段以 `Comment` 标注逻辑外键方向：

| 表 | 字段 | 逻辑外键 | 说明 |
| --- | --- | --- | --- |
| `messages` | `conversation_id` | → `conversations.biz_id` | 消息所属会话 |
| `messages` | `sender_id` | → `users.biz_id` | `sender_type=1`（用户）时有效；系统条目为 0 |
| `messages` | `reply_to_msg_id` | → `messages.biz_id` | 同会话内被引用消息，0 表示无引用 |
| `conversations` | `workspace_id` | → `workspaces.biz_id` | 会话所属工作区 |
| `conversations` | `creator_id` | → `users.biz_id` | 会话创建者 |

应用层在 `SendMessage` 中校验 `reply_to_msg_id` 存在且同会话；成员关系经 `conversation_member` 唯一索引约束。

## 2. 索引现状与评估（阶段 A）

`messages` 现有索引：

- `(conversation_id, seq)` UNIQUE —— Timeline 读扩散主路径（按会话拉取、幂等 seq）。
- `(conversation_id, client_msg_id)` UNIQUE —— 发送幂等。
- `(conversation_id, created_at)` —— 按时间范围拉取。

`conversations` 现有索引：`(workspace_id, type)`、`(last_msg_at)`。

**关于 `sender_id` / `creator_id` 索引：** 阶段 A **不添加**。
当前没有任何按 `sender_id`（“我发的消息”）或 `creator_id`（“我创建的会话”）过滤的查询用例，
仓储仅在写入时设置这两个字段。为避免无用索引拖累写入吞吐与存储，按 YAGNI 不提前建立。
当后续真正引入此类查询时，再补 `(conversation_id, sender_id)` 或 `(creator_id)` 索引并配套防漂移测试。

## 3. 分区与归档（阶段 A 不实施，阶段 B 演进方向）

阶段 A 单实例 + 单 MySQL，消息量可控，**不建分区表**，保持单表 + 上述索引。

阶段 B 当单会话/全局消息量增长后，按以下方向演进：

1. **按 `conversation_id` 分区**：高基数会话天然可哈希/范围分区，Timeline 查询始终带 `conversation_id`，分区裁剪有效。
2. **冷热分层**：以 `created_at` 为界，热数据（近期）留主表，冷数据归档到归档表或对象存储；
   读路径优先热表，命中冷数据时回源归档。归档迁移以 `(conversation_id, created_at)` 现有索引为游标，批量搬迁、可重入。
3. **宽列演进**：对接阶段 B 的 content blocks / metadata 宽列化，`content`、`metadata` 已是 JSON 字段，可平滑迁移到列族存储而不破坏 Timeline 契约。

迁移前置条件：双写或在线迁移工具 + 回滚路径；破坏性变更需先出方案。
