// Package result 定义 media 模块用例出参（CQRS Result）。
package result

// PresignResult 是预签名上传结果。
type PresignResult struct {
	AttachmentID int64  `json:"attachmentId"`
	ObjectKey    string `json:"objectKey"`
	UploadURL    string `json:"uploadUrl"`
}

// AttachmentResult 是附件视图。
type AttachmentResult struct {
	ID          int64  `json:"id"`
	ObjectKey   string `json:"objectKey"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Status      int8   `json:"status"`
	PublicURL   string `json:"publicUrl"`
}
