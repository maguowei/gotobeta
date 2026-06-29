// Package authz 定义鉴权端口（共享内核）。
//
// 业务模块通过 Checker 端口做权限裁决，具体实现（组合 RBAC + ACL + DataScope）
// 在 workspace 模块的基础设施层，组合根注入。该包只声明契约，不含实现，
// 保持框架无关与跨模块可复用。
package authz

import "context"

// Subject 是发起鉴权请求的主体。
type Subject struct {
	UserID int64
}

// Request 描述一次鉴权请求：在哪个工作区、谁、做什么动作、作用于哪个资源实例。
// ResourceType/ResourceID 可空（仅做工作区级动作授权时不填实例）。
type Request struct {
	WorkspaceID  int64
	Subject      Subject
	Action       string // 权限编码，如 message.send、channel.create
	ResourceType string // 资源类型，如 conversation，可空
	ResourceID   string // 资源实例 ID，可空
}

// Checker 裁决鉴权请求。允许返回 nil；拒绝返回 apperr.Forbidden；
// 基础设施异常返回包装后的内部错误。
type Checker interface {
	Check(ctx context.Context, req Request) error
}
