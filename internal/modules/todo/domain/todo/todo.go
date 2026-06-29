package todo

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// Todo 是演示业务实体（聚合根）。
type Todo struct {
	id        int64
	title     Title
	status    Status
	version   int
	createdAt time.Time
	updatedAt time.Time
}

func (t *Todo) ID() int64            { return t.id }
func (t *Todo) Title() Title         { return t.title }
func (t *Todo) Status() Status       { return t.status }
func (t *Todo) Version() int         { return t.version }
func (t *Todo) CreatedAt() time.Time { return t.createdAt }
func (t *Todo) UpdatedAt() time.Time { return t.updatedAt }

// New 创建新的 Todo。
func New(id int64, title string) (*Todo, error) {
	titleVO, err := NewTitle(title)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	t := &Todo{
		id:        id,
		title:     titleVO,
		status:    StatusPending,
		version:   1,
		createdAt: now,
		updatedAt: now,
	}
	return t, nil
}

// UnmarshalFromDB 从数据库记录重建实体，跳过业务校验。仅供 infra 层使用。
func UnmarshalFromDB(id int64, title string, status Status, version int, createdAt, updatedAt time.Time) *Todo {
	return &Todo{
		id:        id,
		title:     Title{value: title},
		status:    status,
		version:   version,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// Complete 将待办标记为完成。
func (t *Todo) Complete() error {
	if t.status == StatusDone {
		return apperr.InvalidParam("待办已完成")
	}
	t.status = StatusDone
	t.updatedAt = time.Now()
	return nil
}

// UpdateTitle 修改待办标题。
func (t *Todo) UpdateTitle(title string) error {
	titleVO, err := NewTitle(title)
	if err != nil {
		return err
	}
	t.title = titleVO
	t.updatedAt = time.Now()
	return nil
}

// IsDone 返回是否已完成。
func (t *Todo) IsDone() bool {
	return t.status == StatusDone
}
