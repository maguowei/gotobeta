// Package response 定义 media 模块的 HTTP 响应体，仅从 application 结果映射。
package response

import mediaresult "github.com/maguowei/gotobeta/internal/modules/media/application/result"

// PresignResponse 是预签名上传响应。
type PresignResponse struct {
	AttachmentID int64  `json:"attachmentId,string"`
	ObjectKey    string `json:"objectKey"`
	UploadURL    string `json:"uploadUrl"`
}

// ToPresignResponse 转换预签名结果。
func ToPresignResponse(out *mediaresult.PresignResult) PresignResponse {
	return PresignResponse{
		AttachmentID: out.AttachmentID,
		ObjectKey:    out.ObjectKey,
		UploadURL:    out.UploadURL,
	}
}

// AttachmentResponse 是附件响应。
type AttachmentResponse struct {
	AttachmentID int64  `json:"attachmentId,string"`
	ObjectKey    string `json:"objectKey"`
	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType"`
	SizeBytes    int64  `json:"sizeBytes"`
	Status       int8   `json:"status"`
	PublicURL    string `json:"publicUrl"`
}

// ToAttachmentResponse 转换附件结果。
func ToAttachmentResponse(out *mediaresult.AttachmentResult) AttachmentResponse {
	return AttachmentResponse{
		AttachmentID: out.ID,
		ObjectKey:    out.ObjectKey,
		FileName:     out.FileName,
		ContentType:  out.ContentType,
		SizeBytes:    out.SizeBytes,
		Status:       out.Status,
		PublicURL:    out.PublicURL,
	}
}
