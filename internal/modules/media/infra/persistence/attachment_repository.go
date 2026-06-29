// Package persistence 是 media 模块的仓储 Ent 实现。
package persistence

import (
	"context"
	"log/slog"

	"github.com/maguowei/gotobeta/internal/ent"
	entattachment "github.com/maguowei/gotobeta/internal/ent/attachment"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/modules/media/domain/attachment"
)

// AttachmentRepository 是附件仓储的 Ent 实现。
type AttachmentRepository struct {
	client *ent.Client
	logger *slog.Logger
}

// NewAttachmentRepository 创建仓储。
func NewAttachmentRepository(client *ent.Client, logger *slog.Logger) *AttachmentRepository {
	return &AttachmentRepository{client: client, logger: logger}
}

// Create 保存附件。
func (r *AttachmentRepository) Create(ctx context.Context, a *attachment.Attachment) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	_, err := client.Attachment.Create().
		SetBizID(a.ID()).
		SetWorkspaceID(a.WorkspaceID()).
		SetUploaderID(a.UploaderID()).
		SetObjectKey(a.ObjectKey()).
		SetFileName(a.FileName()).
		SetContentType(a.ContentType()).
		SetSizeBytes(a.SizeBytes()).
		SetStatus(int8(a.Status())).
		SetMetadata(a.Metadata()).
		SetCreatedAt(a.CreatedAt()).
		SetUpdatedAt(a.UpdatedAt()).
		Save(ctx)
	if err != nil {
		return err
	}
	return nil
}

// FindByID 按业务 ID 查找。
func (r *AttachmentRepository) FindByID(ctx context.Context, id int64) (*attachment.Attachment, error) {
	client := entdb.ClientFromCtx(ctx, r.client)
	row, err := client.Attachment.Query().Where(entattachment.BizID(id)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, attachment.ErrNotFound
		}
		return nil, err
	}
	return attachmentToEntity(row), nil
}

// Save 更新附件可变字段（状态）。
func (r *AttachmentRepository) Save(ctx context.Context, a *attachment.Attachment) error {
	client := entdb.ClientFromCtx(ctx, r.client)
	affected, err := client.Attachment.Update().
		Where(entattachment.BizID(a.ID())).
		SetStatus(int8(a.Status())).
		SetMetadata(a.Metadata()).
		SetUpdatedAt(a.UpdatedAt()).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return attachment.ErrNotFound
	}
	return nil
}

func attachmentToEntity(row *ent.Attachment) *attachment.Attachment {
	return attachment.UnmarshalFromDB(
		row.BizID, row.WorkspaceID, row.UploaderID, row.ObjectKey, row.FileName,
		row.ContentType, row.SizeBytes, attachment.Status(row.Status), row.Metadata,
		row.CreatedAt, row.UpdatedAt,
	)
}
