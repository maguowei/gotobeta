package service

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	mediacmd "github.com/maguowei/gotobeta/internal/modules/media/application/command"
	"github.com/maguowei/gotobeta/internal/modules/media/domain/attachment"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
)

type allowChecker struct{}

func (allowChecker) Check(context.Context, authz.Request) error { return nil }

type denyChecker struct{}

func (denyChecker) Check(context.Context, authz.Request) error { return apperr.Forbidden("denied") }

type fakeIDGen struct{ n int64 }

func (g *fakeIDGen) NextID(context.Context) (int64, error) { return atomic.AddInt64(&g.n, 1), nil }

type stubPresigner struct{}

func (stubPresigner) PresignPut(context.Context, string, time.Duration) (string, error) {
	return "https://obj.example.com/put?sig=x", nil
}
func (stubPresigner) PublicURL(key string) string { return "https://cdn.example.com/" + key }

type memAttachRepo struct {
	items map[int64]*attachment.Attachment
}

func newMemAttachRepo() *memAttachRepo {
	return &memAttachRepo{items: map[int64]*attachment.Attachment{}}
}

func (r *memAttachRepo) Create(_ context.Context, a *attachment.Attachment) error {
	r.items[a.ID()] = a
	return nil
}

func (r *memAttachRepo) FindByID(_ context.Context, id int64) (*attachment.Attachment, error) {
	if a, ok := r.items[id]; ok {
		return a, nil
	}
	return nil, attachment.ErrNotFound
}

func (r *memAttachRepo) Save(_ context.Context, a *attachment.Attachment) error {
	r.items[a.ID()] = a
	return nil
}

func newService(checker authz.Checker) (*AttachmentService, *memAttachRepo) {
	repo := newMemAttachRepo()
	svc := NewAttachmentService(repo, stubPresigner{}, checker, &fakeIDGen{}, time.Minute, slog.Default())
	return svc, repo
}

func TestPresignCreatesPendingAndURL(t *testing.T) {
	svc, repo := newService(allowChecker{})
	out, err := svc.Presign(context.Background(), mediacmd.PresignAttachmentCommand{
		WorkspaceID: 1, UploaderID: 9, FileName: "a.png", ContentType: "image/png", SizeBytes: 100,
	})
	if err != nil {
		t.Fatalf("presign 失败: %v", err)
	}
	if out.UploadURL == "" || out.ObjectKey == "" {
		t.Fatalf("应返回 URL 与 key: %+v", out)
	}
	att := repo.items[out.AttachmentID]
	if att == nil || att.Status() != attachment.StatusPending {
		t.Fatalf("应创建待提交附件: %+v", att)
	}
}

func TestPresignDeniedByChecker(t *testing.T) {
	svc, _ := newService(denyChecker{})
	_, err := svc.Presign(context.Background(), mediacmd.PresignAttachmentCommand{
		WorkspaceID: 1, UploaderID: 9, FileName: "a.png", ContentType: "image/png",
	})
	if err == nil {
		t.Fatal("无权限应被拒绝")
	}
}

func TestCommitByUploaderOnly(t *testing.T) {
	svc, _ := newService(allowChecker{})
	out, _ := svc.Presign(context.Background(), mediacmd.PresignAttachmentCommand{
		WorkspaceID: 1, UploaderID: 9, FileName: "a.png", ContentType: "image/png",
	})
	// 他人提交被拒。
	if _, err := svc.Commit(context.Background(), mediacmd.CommitAttachmentCommand{
		WorkspaceID: 1, OperatorID: 999, AttachmentID: out.AttachmentID,
	}); err == nil {
		t.Fatal("非上传者提交应被拒绝")
	}
	// 本人提交成功。
	res, err := svc.Commit(context.Background(), mediacmd.CommitAttachmentCommand{
		WorkspaceID: 1, OperatorID: 9, AttachmentID: out.AttachmentID,
	})
	if err != nil {
		t.Fatalf("本人提交应成功: %v", err)
	}
	if res.Status != int8(attachment.StatusCommitted) || res.PublicURL == "" {
		t.Fatalf("提交结果错误: %+v", res)
	}
}

func TestCommitNotFound(t *testing.T) {
	svc, _ := newService(allowChecker{})
	_, err := svc.Commit(context.Background(), mediacmd.CommitAttachmentCommand{
		WorkspaceID: 1, OperatorID: 9, AttachmentID: 123456,
	})
	if err == nil {
		t.Fatal("不存在的附件应报错")
	}
	var de *apperr.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("应为 DomainError, got %v", err)
	}
}
