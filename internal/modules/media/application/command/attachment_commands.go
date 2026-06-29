// Package command 定义 media 模块写用例入参（CQRS Command）。
package command

// PresignAttachmentCommand 申请附件预签名上传入参。
type PresignAttachmentCommand struct {
	WorkspaceID int64
	UploaderID  int64
	FileName    string
	ContentType string
	SizeBytes   int64
}

// CommitAttachmentCommand 确认附件上传完成入参。
type CommitAttachmentCommand struct {
	WorkspaceID  int64
	OperatorID   int64
	AttachmentID int64
}
