package request

import "testing"

func TestRequestToCommand(t *testing.T) {
	p := PresignAttachmentRequest{WorkspaceID: 1, FileName: "a.png", ContentType: "image/png", SizeBytes: 10}.ToCommand(9)
	if p.WorkspaceID != 1 || p.UploaderID != 9 || p.FileName != "a.png" {
		t.Fatalf("presign 映射错误: %+v", p)
	}
	c := CommitAttachmentRequest{WorkspaceID: 1}.ToCommand(9, 5001)
	if c.WorkspaceID != 1 || c.OperatorID != 9 || c.AttachmentID != 5001 {
		t.Fatalf("commit 映射错误: %+v", c)
	}
}
