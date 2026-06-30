# gotobeta 对标《企业级 Slack 形态 IM 调研报告》:全维度对比与改进建议

> 状态:对比分析与改进 backlog
> 日期:2026-06-30
> 对标对象:Claude 调研报告《企业级协作 IM 系统(Slack 形态)Go 生产级实现方案与最佳实践》
> 核实基准:本仓库源码 + 两份设计文档(`docs/superpowers/specs/2026-06-29-ai-native-im-backend-design.md`、`docs/superpowers/specs/2026-06-29-im-production-evolution-design.md`)

## 0. 结论摘要

gotobeta 已不是"IM 雏形",而是一个 **Phase 1(M1–M7)已完成、Phase A 生产加固大部分已落地、Phase B 水平扩展已有纲要**的成熟 IM 后端。其设计本身就遵循与调研报告同源的 `im-best-practices`(seq 模型、推拉结合、读写扩散、多端一致),因此两者在核心架构上**高度一致**。

对比的真正价值在于三点:

1. **验证选型一致性** —— 绝大多数维度与报告一致或更优(如未读数推导比报告所称"读扩散未读难算"更简洁)。
2. **暴露有意分歧** —— JSON vs Protobuf、纯读扩散 vs 混合扩散、gorilla vs gobwas、MySQL vs PostgreSQL、搜索路径,均为项目**有意识的阶段化取舍**,多数建议保留。
3. **找出可落地增量** —— 报告有、项目(含路线规划)尚未覆盖且值得现在落地的点:**WS 协议版本协商**(已实现,见 §四)。

## 1. 全维度逐项对比

图例:✅ 已实现 / 🟡 已实现但与报告选型不同 / 🟦 已规划(Phase A/B) / ❌ 未做且未规划 / ⚪ 双方一致放弃

### 1.1 整体架构

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| 单体起步、按域演进微服务 | 单体 Go binary,DDD 分包,显式声明 Phase A 单实例 / Phase B 拆分 | ✅ 完全一致 |
| 控制面/实时面分离(Slack thinkers/talkers) | 可靠路径(HTTP+seq)与实时路径(WS signal)分离(决策 D) | ✅ 思想一致 |
| DDD 限界上下文(IAM/Workspace/Messaging/...) | workspace / messaging / realtime / media + user,聚合分包,`make test-architecture` 守护边界 | ✅ 一致且有自动校验 |
| 服务间 gRPC + MQ | 单实例:进程内 EventBus;Phase B → Redis Streams/Kafka | 🟦 同进程,演进口已留 |
| "先持久化再广播"崩溃安全 | 发送 TX 内落库 → 发 `MessageCreated` → dispatcher 扇出 signal | ✅ 一致 |

### 1.2 自定义协议(传输层)

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| WS 之上承载二进制帧(非裸 TCP) | WS(gorilla)承载 JSON 帧 | ✅ 传输形态一致 |
| **payload 用 Protobuf**(报告:"最难回退,第一天做对") | **JSON 帧**,Protobuf 列为 Phase B | 🟡 关键分歧(见 §2.1) |
| 三语义:req/resp、push、sub | HTTP 承担 req/resp;WS 承担 push(signal/presence/typing/read);无显式 sub(按会话成员隐式投递) | ✅ 语义齐备(分布不同) |
| **版本协商**(首帧带 version,不兼容降级/拒绝) | 已补齐:`GET /ws?v=` 握手协商,`auth_ok.pv` 回带服务端版本,不兼容按 close 码拒绝 | ✅ 已落地(见 §四) |
| 心跳/重连/连接迁移 | 自适应 ping/pong、断线重连后 HTTP 按 last_seq 补拉、ticket 绑定 | ✅ 一致 |

### 1.3 长连接网关

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| C1000K:gobwas/ws + epoll(避免 goroutine-per-conn) | gorilla/websocket,goroutine-per-conn,进程内 Hub | 🟡 单实例够用;Phase B 独立网关(见 §2.3) |
| 连接↔用户/设备映射 | 进程内 Hub `map[userID][]Conn` | ✅(单实例) |
| 全局路由表 `user->gateway`(Redis) | 未做(单实例无需);Phase B 引入 | 🟦 |
| 优雅关闭/过载保护/限流背压 | **已落地**:连接上限(1013 拒绝)、缓冲溢出主动断连+指标(不静默丢)、优雅 close 帧、超时配置化 | ✅ 已超前于演进文档缺口清单 |

### 1.4 消息系统核心

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| 全局 ID 用 Snowflake bigint | localid Snowflake int64(`internal/infra/localid`) | ✅ 一致 |
| per-channel 单调递增 seq | 每会话 `last_seq`,发送 TX 内 `UPDATE...last_seq+1` 行锁分配 | ✅ 一致(报告建议 Redis INCR,项目 DB 行锁,Phase B 转 Redis) |
| seq **连续**递增以便客户端检测 gap | seq 从 1 连续,客户端按连续性补拉 | ✅ 一致 |
| **混合扩散**(DM/小群写扩散 + 大群读扩散) | **纯读扩散统一 timeline**(决策 A) | 🟡 分歧(见 §2.2) |
| 收件箱/timeline 模型 | 读扩散统一 timeline,成员持 read_seq | ✅(单库) |
| 未读数 | `unread = last_seq − read_seq` 推导,不双写计数器 | ✅ 比报告"读扩散未读难算"更简洁 |
| 可靠投递:push+pull+seq 对齐、ack、去重 | signal(尽力)+HTTP 拉取(可靠)+seq 对齐;`client_msg_id` 幂等+唯一索引去重 | ✅ 变体:仅推信号+拉正文,无需逐条 ACK |
| 多端同步/漫游 | read_seq 账号级跨端共享;全量 timeline 漫游 | ✅ 一致 |
| 已读/送达回执、typing | read 回执已实现;typing 走 ephemeral 不落库;送达回执未做(拉取模型下弱需求) | ✅ 基本一致 |

### 1.5 存储设计

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| **PostgreSQL** + 声明式分区 + pg_partman | **MySQL** + Ent + 逻辑外键(固定栈) | 🟡 栈分歧(见 §2.4) |
| 按 created_at 范围分区、冷热分离 | `idx_conv_created` 已为分区预留;Phase A8 文档化迁移路径(`docs/tech/im-storage-partition-archive.md`) | 🟦 预留+文档,未强制建分区 |
| 主键 Snowflake bigint(避免 UUIDv4) | Snowflake bigint,`biz_id` 对外、内部 id 分离 | ✅ 一致 |
| 大频道读扩散查询优化(`channel_seq > ?`) | `uk(conversation_id,seq)` 命中增量拉取 | ✅ 一致 |
| 分库分表(按 workspace/channel) | 单实例未分片;Phase B 演进 | 🟦 |
| pgx/pgxpool + 读写分离 | go-sql-driver/mysql + Ent 连接池;未做读写分离 | 🟡 栈对应,读副本未做 |

### 1.6 Redis 使用

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| presence TTL 心跳续期 | `presence:{ws}:{user}` TTL + writePump 周期 Refresh | ✅ 一致 |
| `user->gateway` 路由表 | 单实例未用;Phase B | 🟦 |
| seq/未读 Redis | seq 走 DB 行锁(Phase 1);未读推导无需缓存 | 🟦 演进口已留 |
| 不用 Redis Pub/Sub 做生产广播,用 MQ | 同进程 EventBus;Phase B 明确 Redis **Streams**/Kafka(非 Pub/Sub) | ✅ 一致 |
| seq 等不可丢键开 AOF | DB 阶段 N/A;Phase B 转 Redis 时需遵循(见 §2 P1) | 🟦 |

### 1.7 其他功能模块

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| JWT+refresh+设备管理+SSO/OAuth+RBAC | JWT+refresh+OAuth+动态 RBAC+ACL+DataScope;JWT 吊销黑名单 Phase A7 | ✅ 一致(RBAC 比报告更完整:落库动态+版本化缓存+审计) |
| 推送 APNs/FCM | ❌ 未做;Phase B(`Pusher` 端口+设备 token 表) | 🟦 |
| 全文搜索(PG FTS→Meili→ES) | `Search` 端口预留,随 AI 语义搜索一起做 | 🟦(路径与报告不同,见 §2.5) |
| 文件/媒体 S3 直传+缩略图 | media 模块 S3 预签名直传;缩略图/预览生成未做 | ✅ 直传已实现,派生资产待补 |
| @提及/通知规则/DND | ❌ 未实现(Phase 1 锁定范围外) | ❌ 功能缺口(见 §2.6) |
| Threads/Reactions/编辑删除置顶 | reply_to_msg_id+is_pinned+撤回(recall)+软删已建;**Thread 聚合/Reaction/编辑 API 未做** | 🟡 模型预留,交互未做 |
| Bot/Slash/Webhook/Events API | `bots` 表+sender_type=bot 已建模,运行时未做(随 AI) | 🟦 |
| 音视频 SFU(Pion/LiveKit) | ❌ 未提及(报告亦标"可选高级") | ❌ 范围外 |
| E2EE 通常不做 | 不做(传输 TLS+静态加密) | ⚪ 一致 |

### 1.8 Go 工程实践

| 报告建议 | gotobeta 现状 | 判定 |
|---|---|---|
| 网络库 gobwas/netpoll;框架 go-zero/Kitex | gorilla/ws + gin(自研 DDD 装配,非框架) | 🟡 选型不同,工程规范完整 |
| DDD 分层 | interfaces/application/domain/infrastructure 严格分层+自动校验 | ✅ 一致且更严格 |
| ctx 贯穿、ants 协程池、有界 channel 背压 | ctx 贯穿;有界 send channel 背压;无 ants(goroutine-per-conn) | ✅ 基本一致 |
| OTel+Prometheus+结构化日志 | OTel trace + Prometheus(含 IM 指标)+ slog + Sentry | ✅ 一致 |
| 压测与容量规划 | ❌ 未见专用压测工具 | ❌ 建议补(见 §2) |

### 1.9 业界参考

报告以 Slack/OpenIM/Centrifugo/Mattermost/Discord/Telegram 为蓝本;gotobeta 设计文档显式参照 OpenIM(seq+推拉)、Mattermost(单体分层)、Telegram(协议分层/session 绑定设备)。✅ 参考系一致。

## 2. 关键选型分歧与决策评注

> 这些是报告与项目**有意分歧**之处,多数项目选择更优或等价,**建议确认保留**,而非盲目改成报告写法。

1. **JSON 帧 vs Protobuf**(报告:第一天做对)。项目把 Protobuf 放 Phase B 有其合理性:实时路径只推**轻量信号**(`{t,cid,seq}`),正文走 HTTP 拉取,Protobuf 的体积/性能收益远低于"WS 推全量正文"的系统。→ **保留 JSON 决策,但补齐报告同样强调的"版本协商"(已落地,见 §四),让未来切 Protobuf 不破坏兼容**。

2. **纯读扩散 vs 混合扩散**。报告建议第一天混合;项目纯读扩散 + `unread=last_seq−read_seq` 推导,**反而规避了报告所称"读扩散未读难算"的痛点**,且多端天然一致。Phase B 已留"按规模压测拐点引入混合"。→ **保留,不提前引入混合(违反简单优先)**。

3. **gorilla vs gobwas/ws(C1000K)**。单实例 goroutine-per-conn 在 Phase A 足够;百万连接是 Phase B 独立网关目标。→ **保留,Phase B 评估 gobwas/netpoll**。

4. **MySQL vs PostgreSQL**。固定栈为 MySQL。报告的 PG 专属建议需翻译为 MySQL 等价物:`PARTITION BY RANGE` 分区、事件调度/外部归档、外部搜索引擎、Snowflake(已用)、Ent 连接池。→ **保留栈,Phase A8 分区文档补充 MySQL 等价方案**。

5. **搜索路径**。报告 PG FTS→Meilisearch→ES;项目跳过 FTS,绑定 AI 语义搜索。→ 合理(AI-Native 定位),`Search` 端口已隔离。

6. **缺失的用户级功能**:@提及/通知偏好/DND、Thread 聚合、Reaction、消息编辑——Phase 1 有意锁定范围外。项目策略是"先加固再演进",**不建议在生产加固未收尾前插入新功能**,列入功能路线 backlog。

## 3. 改进优化建议(优先级)

### P0 — 现在落地(已完成)

- **WS 协议版本协商**(报告 §2.5 / 项目设计文档 §6 的 Protobuf 演进接缝)。详见 §四。

### P1 — 并入现有路线(文档/小补强)

- 报告 §6「seq 等不可丢键转 Redis 后必须开 AOF、不可清除」(OpenIM 踩坑教训)→ 写入 Phase B seq 演进注意事项。
- Phase A8 分区文档补 **MySQL 分区等价方案**(对应报告 PG 分区建议)。
- 媒体派生资产(缩略图/预览生成)纳入 media 模块 backlog。
- 慢消费者背压可选优化:报告 §3.4「先丢 typing/presence 等低优先 ephemeral 帧,signal 帧不可投递才断连」;当前一律断连+补拉已正确,列为可选项(简单优先,暂不实现)。

### P2 — 功能路线 backlog(生产加固收尾后再排)

- @提及 + 通知偏好/DND;Thread 聚合;Reaction;消息编辑。
- 推送 APNs/FCM(Phase B `Pusher` 端口);Bot/Webhook 运行时(随 AI)。
- 压测工具与容量规划(报告 §8)。

## 4. 已落地改进:WS 协议版本协商

**背景**:报告 §2.5 把"协议帧格式 + 版本协商"列为"最难回退、必须第一天做对"的决策;项目设计文档 §6 写了"Protobuf 预留"却未提供启用迁移所需的版本协商机制。补齐版本协商,使未来 JSON→Protobuf 的帧演进不破坏向后兼容,符合项目"以端口/接缝隔离演进点"哲学。

**实现**(改动均在 `internal/modules/realtime/adapter/ws/`,不跨层):

- 契约:`GET /api/v1/ws` 新增可选 query 参数 `v`(协议版本,缺省 1),并声明不兼容时的 close 行为。
- 帧:`Frame` 增 `pv` 字段;`auth_ok` 回带服务端当前版本;新增协议版本常量与不支持版本的应用区间 close 码(4000)。
- 握手:升级后解析 `v`,落在 `[最小支持, 当前]` 范围内则继续并在 `auth_ok` 带 `pv`;超范围按 close 码拒绝,不注册到 Hub。
- 兼容:缺省 `v`(老客户端)按 v1 放行,无回归。

**验证**:`gateway_test.go` 覆盖兼容/不兼容/缺省三路径;`make lint-openapi`、`make test-architecture`、`make verify` 通过。
