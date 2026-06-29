package workspace

import (
	"regexp"
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// slugPattern 约束工作区标识：小写字母、数字、连字符，1-50 位，且不以连字符开头/结尾。
var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,48}[a-z0-9])?$`)

// Workspace 是多租户根聚合。
type Workspace struct {
	id          int64
	slug        string
	name        string
	ownerUserID int64
	status      Status
	settings    map[string]any
	createdAt   time.Time
	updatedAt   time.Time
}

func (w *Workspace) ID() int64                { return w.id }
func (w *Workspace) Slug() string             { return w.slug }
func (w *Workspace) Name() string             { return w.name }
func (w *Workspace) OwnerUserID() int64       { return w.ownerUserID }
func (w *Workspace) Status() Status           { return w.status }
func (w *Workspace) Settings() map[string]any { return w.settings }
func (w *Workspace) CreatedAt() time.Time     { return w.createdAt }
func (w *Workspace) UpdatedAt() time.Time     { return w.updatedAt }

// New 创建工作区，校验 slug 与名称。
func New(id int64, slug, name string, ownerUserID int64) (*Workspace, error) {
	if !slugPattern.MatchString(slug) {
		return nil, apperr.InvalidParam("工作区标识只能包含小写字母、数字和连字符，长度 1-50")
	}
	if name == "" {
		return nil, apperr.InvalidParam("工作区名称不能为空")
	}
	if ownerUserID <= 0 {
		return nil, apperr.InvalidParam("工作区必须有所有者")
	}
	now := time.Now()
	return &Workspace{
		id:          id,
		slug:        slug,
		name:        name,
		ownerUserID: ownerUserID,
		status:      StatusActive,
		settings:    map[string]any{},
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// UnmarshalFromDB 从数据库记录重建聚合，跳过业务校验。仅供 infra 层使用。
func UnmarshalFromDB(id int64, slug, name string, ownerUserID int64, status Status, settings map[string]any, createdAt, updatedAt time.Time) *Workspace {
	if settings == nil {
		settings = map[string]any{}
	}
	return &Workspace{
		id:          id,
		slug:        slug,
		name:        name,
		ownerUserID: ownerUserID,
		status:      status,
		settings:    settings,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// Rename 修改工作区名称。
func (w *Workspace) Rename(name string) error {
	if name == "" {
		return apperr.InvalidParam("工作区名称不能为空")
	}
	w.name = name
	w.updatedAt = time.Now()
	return nil
}

// Disable 停用工作区。
func (w *Workspace) Disable() {
	w.status = StatusDisabled
	w.updatedAt = time.Now()
}
