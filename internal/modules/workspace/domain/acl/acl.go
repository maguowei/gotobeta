// Package acl 是实例级例外授权聚合：对具体资源做显式允许/拒绝。
//
// ACL 是 RBAC 之上的例外补丁，必须可审计、可过期、可回收。
// 裁决优先级：显式拒绝 > 显式允许 > 回落 RBAC。
package acl

import (
	"context"
	"errors"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// ErrNotFound 表示 ACL 记录不存在。
var ErrNotFound = errors.New("acl: not found")

// 主体类型。
const (
	SubjectUser int8 = 1
	SubjectRole int8 = 2
)

// Effect 表示授权效果。
type Effect int8

const (
	// EffectAllow 显式允许。
	EffectAllow Effect = 1
	// EffectDeny 显式拒绝。
	EffectDeny Effect = 2
)

// Entry 是一条 ACL 例外授权。
type Entry struct {
	id           int64
	workspaceID  int64
	subjectType  int8
	subjectID    int64
	resourceType string
	resourceID   string
	actionCode   string
	effect       Effect
	reason       string
	sourceType   int8
	expiresAt    *time.Time
	createdBy    int64
	createdAt    time.Time
}

// NewEntry 创建 ACL 例外授权，强制 reason 非空（避免黑箱）。
func NewEntry(id, workspaceID int64, subjectType int8, subjectID int64, resourceType, resourceID, actionCode string, effect Effect, reason string, createdBy int64, expiresAt *time.Time) (*Entry, error) {
	if reason == "" {
		return nil, apperr.InvalidParam("ACL 授权必须填写原因")
	}
	if effect != EffectAllow && effect != EffectDeny {
		return nil, apperr.InvalidParam("ACL 授权效果非法")
	}
	return &Entry{
		id:           id,
		workspaceID:  workspaceID,
		subjectType:  subjectType,
		subjectID:    subjectID,
		resourceType: resourceType,
		resourceID:   resourceID,
		actionCode:   actionCode,
		effect:       effect,
		reason:       reason,
		sourceType:   1,
		expiresAt:    expiresAt,
		createdBy:    createdBy,
		createdAt:    time.Now(),
	}, nil
}

// UnmarshalEntry 从数据库重建。
func UnmarshalEntry(id, workspaceID int64, subjectType int8, subjectID int64, resourceType, resourceID, actionCode string, effect Effect, reason string, sourceType int8, expiresAt *time.Time, createdBy int64, createdAt time.Time) *Entry {
	return &Entry{
		id: id, workspaceID: workspaceID, subjectType: subjectType, subjectID: subjectID,
		resourceType: resourceType, resourceID: resourceID, actionCode: actionCode,
		effect: effect, reason: reason, sourceType: sourceType, expiresAt: expiresAt,
		createdBy: createdBy, createdAt: createdAt,
	}
}

func (e *Entry) ID() int64          { return e.id }
func (e *Entry) Effect() Effect     { return e.effect }
func (e *Entry) ActionCode() string { return e.actionCode }

// IsActive 返回该条目在给定时刻是否有效（未过期）。
func (e *Entry) IsActive(now time.Time) bool {
	return e.expiresAt == nil || e.expiresAt.After(now)
}

// Repository 定义 ACL 仓储接口。
type Repository interface {
	Grant(ctx context.Context, e *Entry) error
	Revoke(ctx context.Context, id int64) error
	// FindDecisive 返回作用于 (subject 或其角色, resource, action) 的有效裁决条目。
	// 实现需保证拒绝优先：若存在有效 Deny 则返回该 Deny。
	FindDecisive(ctx context.Context, workspaceID int64, userID int64, roleIDs []int64, resourceType, resourceID, actionCode string, now time.Time) (*Entry, error)
}

// Decide 根据 RBAC 结果与 ACL 裁决条目给出最终决定。
// decisive 为 nil 时回落 RBAC；否则 Deny 拒绝、Allow 放行。
func Decide(rbacAllowed bool, decisive *Entry) bool {
	if decisive != nil {
		return decisive.Effect() == EffectAllow
	}
	return rbacAllowed
}
