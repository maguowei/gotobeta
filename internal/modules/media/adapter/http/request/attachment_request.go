// Package request 定义 media 模块的 HTTP 请求体，仅从 application 命令映射。
package request

import mediacmd "github.com/maguowei/gotobeta/internal/modules/media/application/command"

// PresignAttachmentRequest 申请预签名上传请求。
type PresignAttachmentRequest struct {
	WorkspaceID int64  `json:"workspaceId,string" binding:"required"`
	FileName    string `json:"fileName" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	SizeBytes   int64  `json:"sizeBytes"`
}

// ToCommand 转换为命令。
func (r PresignAttachmentRequest) ToCommand(uploaderID int64) mediacmd.PresignAttachmentCommand {
	return mediacmd.PresignAttachmentCommand{
		WorkspaceID: r.WorkspaceID,
		UploaderID:  uploaderID,
		FileName:    r.FileName,
		ContentType: r.ContentType,
		SizeBytes:   r.SizeBytes,
	}
}

// CommitAttachmentRequest 确认上传请求。
type CommitAttachmentRequest struct {
	WorkspaceID int64 `json:"workspaceId,string" binding:"required"`
}

// ToCommand 转换为命令。
func (r CommitAttachmentRequest) ToCommand(operatorID, attachmentID int64) mediacmd.CommitAttachmentCommand {
	return mediacmd.CommitAttachmentCommand{
		WorkspaceID:  r.WorkspaceID,
		OperatorID:   operatorID,
		AttachmentID: attachmentID,
	}
}
