// Package refreshtoken 定义 Refresh Token 的支撑性持久化模型。
//
// 本包不是聚合根：撤销 / 轮换的并发安全由 repository 的原子条件 UPDATE 保证，
// 没有需要在领域内存中强制的不变量，因此状态字段保持公开，只用 New 收敛构造校验。
//
// 构件清单：
//   - RefreshToken 支撑性持久化模型——refresh token 记录（refreshtoken.go）
//   - New          构造工厂——收敛构造校验（refreshtoken.go）
//   - Repository   仓储接口（repository.go）
//   - ErrNotFound  哨兵错误（errors.go）
package refreshtoken
