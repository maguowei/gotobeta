// Package actiontoken 定义一次性动作 Token 的支撑性持久化模型。
//
// 本包不是聚合根：一次性消费的并发安全由 repository 的原子条件 UPDATE 保证，
// 没有需要在领域内存中强制的不变量，因此状态字段保持公开，只用 New 收敛构造校验。
//
// 构件清单：
//   - ActionToken   支撑性持久化模型——一次性动作 token 记录（actiontoken.go）
//   - New           构造工厂——收敛构造校验（actiontoken.go）
//   - IsValidPurpose purpose 合法性判断（actiontoken.go）
//   - Repository    仓储接口（repository.go）
//   - ErrNotFound   哨兵错误（errors.go）
package actiontoken
