// Package oauthstate 定义 OAuth 登录流程的支撑性持久化模型。
//
// 本包不是聚合根：OAuthState 是纯技术性 CSRF 短时凭据，无业务不变量，
// 生命周期（创建/消费/过期）完全由基础设施驱动，因此保持公开字段。
//
// 构件清单：
//   - OAuthState 支撑性持久化模型——OAuth state 记录（oauthstate.go）
//   - Profile    值对象——三方平台用户资料（profile.go）
//   - Repository 仓储接口（repository.go）
//   - ErrNotFound 哨兵错误（errors.go）
package oauthstate
