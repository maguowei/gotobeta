package response

import (
	"testing"

	mediaresult "github.com/maguowei/gotobeta/internal/modules/media/application/result"
)

func TestResponses(t *testing.T) {
	p := ToPresignResponse(&mediaresult.PresignResult{AttachmentID: 5001, ObjectKey: "k", UploadURL: "u"})
	if p.AttachmentID != 5001 || p.UploadURL != "u" {
		t.Fatalf("presign 响应错误: %+v", p)
	}
	a := ToAttachmentResponse(&mediaresult.AttachmentResult{ID: 5001, FileName: "a.png", Status: 2, PublicURL: "url"})
	if a.AttachmentID != 5001 || a.Status != 2 || a.PublicURL != "url" {
		t.Fatalf("附件响应错误: %+v", a)
	}
}
