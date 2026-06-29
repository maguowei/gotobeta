// Package event 定义领域事件的发布端口与基础类型。
//
// 该包是共享内核（internal/pkg）的一部分，只声明契约：
// 业务模块产生实现 Event 的领域事件，application 层经 Publisher 发布，
// 具体派发由基础设施层（internal/infra/eventbus）实现并在组合根注入。
package event

import (
	"context"
	"time"
)

// Event 是领域事件的最小契约。实现者需提供稳定的事件名与发生时间。
type Event interface {
	// Name 返回事件名（用于订阅路由），同类事件应返回稳定常量。
	Name() string
	// OccurredAt 返回事件发生的服务端时间。
	OccurredAt() time.Time
}

// Handler 处理单个领域事件。返回的 error 由总线记录，不应阻断其他订阅者。
type Handler func(ctx context.Context, e Event) error

// Publisher 发布领域事件。推送是尽力而为：发布失败不应破坏主业务事务，
// 调用方应在事务提交后再发布。
type Publisher interface {
	Publish(ctx context.Context, events ...Event) error
}

// BaseEvent 提供 Event 的公共字段，供具体事件内嵌复用。
type BaseEvent struct {
	name       string
	occurredAt time.Time
}

// NewBaseEvent 构造带名称与发生时间的基础事件。
func NewBaseEvent(name string, occurredAt time.Time) BaseEvent {
	return BaseEvent{name: name, occurredAt: occurredAt}
}

// Name 返回事件名。
func (e BaseEvent) Name() string { return e.name }

// OccurredAt 返回事件发生时间。
func (e BaseEvent) OccurredAt() time.Time { return e.occurredAt }
