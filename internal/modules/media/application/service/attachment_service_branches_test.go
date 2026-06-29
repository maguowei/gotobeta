package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	mediacmd "github.com/maguowei/gotobeta/internal/modules/media/application/command"
	mediaport "github.com/maguowei/gotobeta/internal/modules/media/application/port"
	"github.com/maguowei/gotobeta/internal/modules/media/domain/attachment"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
	"github.com/maguowei/gotobeta/internal/pkg/authz"
	"github.com/maguowei/gotobeta/internal/pkg/idgen"
)

// errIDGen 注入 ID 生成失败。
type errIDGen struct{}

func (errIDGen) NextID(context.Context) (int64, error) { return 0, errors.New("boom") }

// errPresigner 注入预签名失败，但 PublicURL 正常。
type errPresigner struct{}

func (errPresigner) PresignPut(context.Context, string, time.Duration) (string, error) {
	return "", errors.New("presign boom")
}
func (errPresigner) PublicURL(key string) string { return "https://cdn.example.com/" + key }

// errRepo 在指定方法注入失败，其余委托内存实现。
type errRepo struct {
	*memAttachRepo
	createErr error
	findErr   error // 非 ErrNotFound 的查询错误
	saveErr   error
}

func (r *errRepo) Create(ctx context.Context, a *attachment.Attachment) error {
	if r.createErr != nil {
		return r.createErr
	}
	return r.memAttachRepo.Create(ctx, a)
}

func (r *errRepo) FindByID(ctx context.Context, id int64) (*attachment.Attachment, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.memAttachRepo.FindByID(ctx, id)
}

func (r *errRepo) Save(ctx context.Context, a *attachment.Attachment) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	return r.memAttachRepo.Save(ctx, a)
}

func presignCmd() mediacmd.PresignAttachmentCommand {
	return mediacmd.PresignAttachmentCommand{
		WorkspaceID: 1, UploaderID: 9, FileName: "a.png", ContentType: "image/png", SizeBytes: 100,
	}
}

func assertDomainErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("应返回错误")
	}
	var de *apperr.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("应为 DomainError, got %v", err)
	}
}

// TestNewAttachmentServiceDefaultTTL 校验非正 TTL 回退默认值。
func TestNewAttachmentServiceDefaultTTL(t *testing.T) {
	t.Parallel()
	svc := NewAttachmentService(newMemAttachRepo(), stubPresigner{}, allowChecker{}, &fakeIDGen{}, 0, slog.Default())
	if svc.presignTTL != 15*time.Minute {
		t.Fatalf("默认 TTL 应为 15m, got %v", svc.presignTTL)
	}
}

// TestPresignNilPresigner 校验未启用对象存储时 Presign 直接报错。
func TestPresignNilPresigner(t *testing.T) {
	t.Parallel()
	var nilPresigner mediaport.Presigner
	svc := NewAttachmentService(newMemAttachRepo(), nilPresigner, allowChecker{}, &fakeIDGen{}, time.Minute, slog.Default())
	_, err := svc.Presign(context.Background(), presignCmd())
	assertDomainErr(t, err)
}

// TestPresignErrorBranches 表驱动覆盖 Presign 各失败分支。
func TestPresignErrorBranches(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		presigner mediaport.Presigner
		idGen     idgen.Generator
		repo      attachment.Repository
		cmd       mediacmd.PresignAttachmentCommand
	}{
		{
			name:      "ID 生成失败",
			presigner: stubPresigner{},
			idGen:     errIDGen{},
			repo:      newMemAttachRepo(),
			cmd:       presignCmd(),
		},
		{
			name:      "附件校验失败（空文件名）",
			presigner: stubPresigner{},
			idGen:     &fakeIDGen{},
			repo:      newMemAttachRepo(),
			cmd:       mediacmd.PresignAttachmentCommand{WorkspaceID: 1, UploaderID: 9, FileName: "", ContentType: "image/png"},
		},
		{
			name:      "仓储 Create 失败",
			presigner: stubPresigner{},
			idGen:     &fakeIDGen{},
			repo:      &errRepo{memAttachRepo: newMemAttachRepo(), createErr: errors.New("db down")},
			cmd:       presignCmd(),
		},
		{
			name:      "预签名生成失败",
			presigner: errPresigner{},
			idGen:     &fakeIDGen{},
			repo:      newMemAttachRepo(),
			cmd:       presignCmd(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := NewAttachmentService(tt.repo, tt.presigner, allowChecker{}, tt.idGen, time.Minute, slog.Default())
			_, err := svc.Presign(context.Background(), tt.cmd)
			assertDomainErr(t, err)
		})
	}
}

// TestCommitFindError 校验非 ErrNotFound 的查询错误被包装为内部错误。
func TestCommitFindError(t *testing.T) {
	t.Parallel()
	repo := &errRepo{memAttachRepo: newMemAttachRepo(), findErr: errors.New("db down")}
	svc := NewAttachmentService(repo, stubPresigner{}, allowChecker{}, &fakeIDGen{}, time.Minute, slog.Default())
	_, err := svc.Commit(context.Background(), mediacmd.CommitAttachmentCommand{WorkspaceID: 1, OperatorID: 9, AttachmentID: 1})
	assertDomainErr(t, err)
}

// TestCommitSaveError 校验状态机切换后保存失败被包装为内部错误。
func TestCommitSaveError(t *testing.T) {
	t.Parallel()
	repo := &errRepo{memAttachRepo: newMemAttachRepo(), saveErr: errors.New("db down")}
	svc := NewAttachmentService(repo, stubPresigner{}, allowChecker{}, &fakeIDGen{}, time.Minute, slog.Default())
	out, err := svc.Presign(context.Background(), presignCmd())
	if err != nil {
		t.Fatalf("presign 失败: %v", err)
	}
	_, err = svc.Commit(context.Background(), mediacmd.CommitAttachmentCommand{WorkspaceID: 1, OperatorID: 9, AttachmentID: out.AttachmentID})
	assertDomainErr(t, err)
}

// TestGet 覆盖 Get 正常返回与不存在分支。
func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("正常返回视图含公共 URL", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService(allowChecker{})
		out, err := svc.Presign(context.Background(), presignCmd())
		if err != nil {
			t.Fatalf("presign 失败: %v", err)
		}
		res, err := svc.Get(context.Background(), out.AttachmentID)
		if err != nil {
			t.Fatalf("Get 失败: %v", err)
		}
		if res.ID != out.AttachmentID || res.PublicURL == "" {
			t.Fatalf("Get 结果错误: %+v", res)
		}
		if res.Status != int8(attachment.StatusPending) {
			t.Fatalf("未提交附件状态应为 Pending: %+v", res)
		}
	})

	t.Run("不存在报错", func(t *testing.T) {
		t.Parallel()
		svc, _ := newService(allowChecker{})
		_, err := svc.Get(context.Background(), 999999)
		assertDomainErr(t, err)
	})

	t.Run("查询底层错误被包装", func(t *testing.T) {
		t.Parallel()
		repo := &errRepo{memAttachRepo: newMemAttachRepo(), findErr: errors.New("db down")}
		svc := NewAttachmentService(repo, stubPresigner{}, allowChecker{}, &fakeIDGen{}, time.Minute, slog.Default())
		_, err := svc.Get(context.Background(), 1)
		assertDomainErr(t, err)
	})
}

// 编译期保证 fake 满足依赖接口。
var (
	_ authz.Checker   = allowChecker{}
	_ idgen.Generator = errIDGen{}
)
