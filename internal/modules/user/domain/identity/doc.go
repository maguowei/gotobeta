// Package identity 定义第三方登录身份的支撑性持久化模型。
//
// 本包不是聚合根：Identity 是第三方平台 profile 的快照记录，无业务不变量
// 与状态机；"解绑需保留至少一种登录方式"等规则属于跨聚合协调，由 application 层负责。
//
// 构件清单：
//   - Identity   支撑性持久化模型——第三方身份记录（identity.go）
//   - Repository 仓储接口（repository.go）
//   - ErrNotFound 哨兵错误（errors.go）
package identity
