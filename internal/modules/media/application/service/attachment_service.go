// Package service 编排 media 模块用例（附件预签名上传与提交）。
package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	mediacmd "github.com/maguowei/gotobeta/internal/modules/media/application/command"
	mediaport "github.com/maguowei/gotobeta/internal/modules/media/application/port"
	mediaresult "github.com/maguowei/gotobeta/internal/modules/media/application/result"
	"github.com/maguowei/gotobeta/internal/modules/media/domain/attachment"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
)

// AttachmentService 编排附件相关用例。
type AttachmentService struct {
	attachments attachment.Repository
	presigner   mediaport.Presigner
	checker     authz.Checker
	idGenerator idgen.Generator
	presignTTL  time.Duration
	logger      *slog.Logger
}

// NewAttachmentService 创建服务。
func NewAttachmentService(
	attachments attachment.Repository,
	presigner mediaport.Presigner,
	checker authz.Checker,
	idGenerator idgen.Generator,
	presignTTL time.Duration,
	logger *slog.Logger,
) *AttachmentService {
	if presignTTL <= 0 {
		presignTTL = 15 * time.Minute
	}
	return &AttachmentService{
		attachments: attachments,
		presigner:   presigner,
		checker:     checker,
		idGenerator: idGenerator,
		presignTTL:  presignTTL,
		logger:      logger,
	}
}

// Presign 申请预签名上传：建待提交附件 + 返回预签名 PUT URL。
func (s *AttachmentService) Presign(ctx context.Context, cmd mediacmd.PresignAttachmentCommand) (*mediaresult.PresignResult, error) {
	if s.presigner == nil {
		return nil, apperr.Internal("对象存储未启用", nil)
	}
	if err := s.checker.Check(ctx, authz.Request{
		WorkspaceID: cmd.WorkspaceID,
		Subject:     authz.Subject{UserID: cmd.UploaderID},
		Action:      authz.PermMessageSend,
	}); err != nil {
		return nil, err
	}

	id, err := s.idGenerator.NextID(ctx)
	if err != nil {
		return nil, apperr.Internal("生成附件 ID 失败", err)
	}
	objectKey := buildObjectKey(cmd.WorkspaceID, id, cmd.FileName)
	att, err := attachment.New(id, cmd.WorkspaceID, cmd.UploaderID, objectKey, cmd.FileName, cmd.ContentType, cmd.SizeBytes)
	if err != nil {
		return nil, err
	}
	if err := s.attachments.Create(ctx, att); err != nil {
		return nil, apperr.Internal("保存附件失败", err)
	}
	uploadURL, err := s.presigner.PresignPut(ctx, objectKey, s.presignTTL)
	if err != nil {
		return nil, apperr.Internal("生成预签名 URL 失败", err)
	}
	s.logger.InfoContext(ctx, "attachment presigned", slog.Int64("attachmentId", id), slog.Int64("workspaceId", cmd.WorkspaceID))
	return &mediaresult.PresignResult{AttachmentID: id, ObjectKey: objectKey, UploadURL: uploadURL}, nil
}

// Commit 确认附件上传完成并置为已提交（仅上传者本人）。
func (s *AttachmentService) Commit(ctx context.Context, cmd mediacmd.CommitAttachmentCommand) (*mediaresult.AttachmentResult, error) {
	att, err := s.attachments.FindByID(ctx, cmd.AttachmentID)
	if err != nil {
		if stderrors.Is(err, attachment.ErrNotFound) {
			return nil, apperr.NotFound("附件不存在")
		}
		return nil, apperr.Internal("查询附件失败", err)
	}
	if att.UploaderID() != cmd.OperatorID {
		return nil, apperr.Forbidden("只能提交自己上传的附件")
	}
	if att.Commit() {
		if err := s.attachments.Save(ctx, att); err != nil {
			return nil, apperr.Internal("更新附件状态失败", err)
		}
	}
	return s.toResult(att), nil
}

// Get 返回附件视图（含公共 URL），供消息引用校验。
func (s *AttachmentService) Get(ctx context.Context, attachmentID int64) (*mediaresult.AttachmentResult, error) {
	att, err := s.attachments.FindByID(ctx, attachmentID)
	if err != nil {
		if stderrors.Is(err, attachment.ErrNotFound) {
			return nil, apperr.NotFound("附件不存在")
		}
		return nil, apperr.Internal("查询附件失败", err)
	}
	return s.toResult(att), nil
}

func (s *AttachmentService) toResult(att *attachment.Attachment) *mediaresult.AttachmentResult {
	publicURL := ""
	if s.presigner != nil {
		publicURL = s.presigner.PublicURL(att.ObjectKey())
	}
	return &mediaresult.AttachmentResult{
		ID:          att.ID(),
		ObjectKey:   att.ObjectKey(),
		FileName:    att.FileName(),
		ContentType: att.ContentType(),
		SizeBytes:   att.SizeBytes(),
		Status:      int8(att.Status()),
		PublicURL:   publicURL,
	}
}

// buildObjectKey 生成稳定对象 key：workspace/<ws>/<attachmentID>/<fileName>。
func buildObjectKey(workspaceID, attachmentID int64, fileName string) string {
	return fmt.Sprintf("workspace/%d/%d/%s", workspaceID, attachmentID, fileName)
}
