// Package todo 定义 Todo 聚合的领域模型。
//
// Todo 聚合根携带 version 字段用于乐观并发控制：version 由 repository.Save
// 以当前版本为更新条件并自增，领域方法不触碰它；版本冲突返回 ErrConflict。
//
// 构件清单：
//   - Todo        聚合根（todo.go）
//   - Title       值对象——待办标题，含构造校验与相等性（title.go）
//   - Status      值对象——待办状态（status.go）
//   - CreatedEvent 领域事件——待办创建（events.go，需启用异步事件能力）
//   - Repository  仓储接口（repository.go）
//   - ErrNotFound / ErrConflict  哨兵错误（errors.go）
package todo
