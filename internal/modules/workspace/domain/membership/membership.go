// Package membership 是工作区成员关系聚合：记录用户属于哪个工作区。
// 角色授权由 rbac 聚合承载，本聚合只管成员存在与状态。
package membership

import (
	"context"
	"errors"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

var (
	// ErrNotFound 表示成员关系不存在。
	ErrNotFound = errors.New("membership: not found")
	// ErrAlreadyMember 表示用户已是该工作区成员。
	ErrAlreadyMember = errors.New("membership: already member")
)

// Status 表示成员状态。
type Status int8

const (
	// StatusActive 正常。
	StatusActive Status = 1
	// StatusDisabled 禁用。
	StatusDisabled Status = 2
)

// Member 是工作区成员关系聚合根。
type Member struct {
	id          int64
	workspaceID int64
	userID      int64
	status      Status
	joinedAt    time.Time
	createdAt   time.Time
	updatedAt   time.Time
}

func (m *Member) ID() int64            { return m.id }
func (m *Member) WorkspaceID() int64   { return m.workspaceID }
func (m *Member) UserID() int64        { return m.userID }
func (m *Member) Status() Status       { return m.status }
func (m *Member) JoinedAt() time.Time  { return m.joinedAt }
func (m *Member) CreatedAt() time.Time { return m.createdAt }
func (m *Member) UpdatedAt() time.Time { return m.updatedAt }

// New 创建成员关系。
func New(id, workspaceID, userID int64) (*Member, error) {
	if workspaceID <= 0 || userID <= 0 {
		return nil, apperr.InvalidParam("成员关系需要有效的工作区与用户")
	}
	now := time.Now()
	return &Member{
		id:          id,
		workspaceID: workspaceID,
		userID:      userID,
		status:      StatusActive,
		joinedAt:    now,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// UnmarshalFromDB 从数据库记录重建聚合。
func UnmarshalFromDB(id, workspaceID, userID int64, status Status, joinedAt, createdAt, updatedAt time.Time) *Member {
	return &Member{
		id:          id,
		workspaceID: workspaceID,
		userID:      userID,
		status:      status,
		joinedAt:    joinedAt,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// Repository 定义成员关系仓储接口。
type Repository interface {
	Add(ctx context.Context, m *Member) error
	FindByWorkspaceUser(ctx context.Context, workspaceID, userID int64) (*Member, error)
	ListByUser(ctx context.Context, userID int64) ([]*Member, error)
	ListByWorkspace(ctx context.Context, workspaceID int64) ([]*Member, error)
}
