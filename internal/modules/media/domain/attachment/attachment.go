// Package attachment 是附件聚合：消息体只存引用 key，元数据与生命周期由本聚合管理。
package attachment

import (
	"time"

	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

// Status 表示附件状态。
type Status int8

const (
	// StatusPending 待提交（已签发预签名，尚未确认上传完成）。
	StatusPending Status = 1
	// StatusCommitted 已提交（被消息引用，确认有效）。
	StatusCommitted Status = 2
)

// Attachment 是附件聚合根。
type Attachment struct {
	id          int64
	workspaceID int64
	uploaderID  int64
	objectKey   string
	fileName    string
	contentType string
	sizeBytes   int64
	status      Status
	metadata    map[string]any
	createdAt   time.Time
	updatedAt   time.Time
}

func (a *Attachment) ID() int64                { return a.id }
func (a *Attachment) WorkspaceID() int64       { return a.workspaceID }
func (a *Attachment) UploaderID() int64        { return a.uploaderID }
func (a *Attachment) ObjectKey() string        { return a.objectKey }
func (a *Attachment) FileName() string         { return a.fileName }
func (a *Attachment) ContentType() string      { return a.contentType }
func (a *Attachment) SizeBytes() int64         { return a.sizeBytes }
func (a *Attachment) Status() Status           { return a.status }
func (a *Attachment) Metadata() map[string]any { return a.metadata }
func (a *Attachment) CreatedAt() time.Time     { return a.createdAt }
func (a *Attachment) UpdatedAt() time.Time     { return a.updatedAt }

// New 创建待提交附件。objectKey 由应用层基于 ID 生成后传入。
func New(id, workspaceID, uploaderID int64, objectKey, fileName, contentType string, sizeBytes int64) (*Attachment, error) {
	if fileName == "" {
		return nil, apperr.InvalidParam("文件名不能为空")
	}
	if contentType == "" {
		return nil, apperr.InvalidParam("内容类型不能为空")
	}
	if sizeBytes < 0 {
		return nil, apperr.InvalidParam("文件大小非法")
	}
	now := time.Now()
	return &Attachment{
		id: id, workspaceID: workspaceID, uploaderID: uploaderID,
		objectKey: objectKey, fileName: fileName, contentType: contentType, sizeBytes: sizeBytes,
		status: StatusPending, metadata: map[string]any{}, createdAt: now, updatedAt: now,
	}, nil
}

// Commit 确认附件已上传并被引用；非待提交状态时幂等返回 false。
func (a *Attachment) Commit() bool {
	if a.status != StatusPending {
		return false
	}
	a.status = StatusCommitted
	a.updatedAt = time.Now()
	return true
}

// UnmarshalFromDB 从数据库重建附件聚合。
func UnmarshalFromDB(id, workspaceID, uploaderID int64, objectKey, fileName, contentType string, sizeBytes int64, status Status, metadata map[string]any, createdAt, updatedAt time.Time) *Attachment {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &Attachment{
		id: id, workspaceID: workspaceID, uploaderID: uploaderID,
		objectKey: objectKey, fileName: fileName, contentType: contentType, sizeBytes: sizeBytes,
		status: status, metadata: metadata, createdAt: createdAt, updatedAt: updatedAt,
	}
}
