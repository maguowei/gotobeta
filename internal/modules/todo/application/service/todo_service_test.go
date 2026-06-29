package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"testing"
	"time"

	todocmd "github.com/maguowei/gotobeta/internal/modules/todo/application/command"
	todoquery "github.com/maguowei/gotobeta/internal/modules/todo/application/query"
	"github.com/maguowei/gotobeta/internal/modules/todo/domain/todo"
	"github.com/maguowei/gotobeta/internal/pkg/apperr"
)

var testLogger = slog.Default()

type stubRepository struct {
	created   *todo.Todo
	saved     *todo.Todo
	deleted   int64
	listItems []*todo.Todo
	findItem  *todo.Todo
	createErr error
	findErr   error
	listErr   error
	saveErr   error
	deleteErr error
}

func (r *stubRepository) Create(_ context.Context, t *todo.Todo) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = t
	return nil
}

func (r *stubRepository) FindByID(_ context.Context, id int64) (*todo.Todo, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.findItem != nil && r.findItem.ID() == id {
		return r.findItem, nil
	}
	return nil, todo.ErrNotFound
}

func (r *stubRepository) Save(_ context.Context, t *todo.Todo) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = t
	return nil
}

func (r *stubRepository) Delete(_ context.Context, id int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = id
	return nil
}

func (r *stubRepository) List(context.Context) ([]*todo.Todo, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listItems, nil
}

type stubIDGenerator struct {
	nextID int64
	err    error
}

func (g *stubIDGenerator) NextID(context.Context) (int64, error) {
	if g.err != nil {
		return 0, g.err
	}
	g.nextID++
	return g.nextID, nil
}

type stubTxRunner struct {
	called bool
}

func (r *stubTxRunner) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	r.called = true
	return fn(ctx)
}

func TestTodoService_CreateTodo(t *testing.T) {
	repo := &stubRepository{}
	txRunner := &stubTxRunner{}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		txRunner,
		testLogger,
	)

	out, err := svc.CreateTodo(context.Background(), todocmd.CreateTodoCommand{Title: "  write tests  "})
	if err != nil {
		t.Fatalf("CreateTodo() error = %v", err)
	}

	if repo.created == nil {
		t.Fatal("repository.created = nil")
	}

	if out.ID == 0 {
		t.Fatal("out.ID = 0, want generated id")
	}

	if out.Title != "write tests" {
		t.Fatalf("out.Title = %q, want write tests", out.Title)
	}

	if !txRunner.called {
		t.Fatal("txRunner.called = false, want create use case to run in transaction")
	}
}

func TestTodoService_CreateTodoFailures(t *testing.T) {
	tests := []struct {
		name     string
		repo     *stubRepository
		idgen    *stubIDGenerator
		cmd      todocmd.CreateTodoCommand
		wantKind apperr.Kind
	}{
		{
			name:     "id generator error",
			repo:     &stubRepository{},
			idgen:    &stubIDGenerator{err: stderrors.New("snowflake down")},
			cmd:      todocmd.CreateTodoCommand{Title: "write tests"},
			wantKind: apperr.KindInternal,
		},
		{
			name:  "empty title",
			repo:  &stubRepository{},
			idgen: &stubIDGenerator{},
			cmd:   todocmd.CreateTodoCommand{Title: " "},
		},
		{
			name:     "repository create error",
			repo:     &stubRepository{createErr: stderrors.New("insert failed")},
			idgen:    &stubIDGenerator{},
			cmd:      todocmd.CreateTodoCommand{Title: "write tests"},
			wantKind: apperr.KindInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTodoService(
				tt.repo,
				tt.idgen,
				&stubTxRunner{},
				testLogger,
			)

			_, err := svc.CreateTodo(context.Background(), tt.cmd)
			if err == nil {
				t.Fatalf("CreateTodo() error = nil, want error")
			}
			if tt.wantKind != 0 {
				assertDomainKind(t, err, tt.wantKind)
			}
		})
	}
}

func TestTodoService_ListTodos(t *testing.T) {
	now := time.Now()
	repo := &stubRepository{
		listItems: []*todo.Todo{
			todo.UnmarshalFromDB(1, "write tests", todo.StatusPending, 1, now, now),
		},
	}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		&stubTxRunner{},
		testLogger,
	)

	items, err := svc.ListTodos(context.Background(), todoquery.ListTodosQuery{})
	if err != nil {
		t.Fatalf("ListTodos() error = %v", err)
	}

	if len(items) != 1 || items[0].Title != "write tests" {
		t.Fatalf("unexpected items: %v", items)
	}
}

func TestTodoService_DoesNotWrapContextCancellationAsDomainError(t *testing.T) {
	repo := &stubRepository{listErr: context.Canceled}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		&stubTxRunner{},
		testLogger,
	)

	_, err := svc.ListTodos(context.Background(), todoquery.ListTodosQuery{})
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("ListTodos() error = %v, want context.Canceled", err)
	}

	var domainErr *apperr.DomainError
	if stderrors.As(err, &domainErr) {
		t.Fatalf("ListTodos() error = %v, should not be DomainError", err)
	}
}

func TestTodoService_ListTodosWrapsInfrastructureError(t *testing.T) {
	repo := &stubRepository{listErr: stderrors.New("select failed")}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		&stubTxRunner{},
		testLogger,
	)

	_, err := svc.ListTodos(context.Background(), todoquery.ListTodosQuery{})
	assertDomainKind(t, err, apperr.KindInternal)
}

func TestTodoService_GetTodo(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		repo     *stubRepository
		wantKind apperr.Kind
	}{
		{
			name: "success",
			repo: &stubRepository{findItem: todo.UnmarshalFromDB(1, "write tests", todo.StatusPending, 1, now, now)},
		},
		{name: "not found", repo: &stubRepository{}, wantKind: apperr.KindNotFound},
		{name: "infra error", repo: &stubRepository{findErr: stderrors.New("select failed")}, wantKind: apperr.KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTodoService(
				tt.repo,
				&stubIDGenerator{},
				&stubTxRunner{},
				testLogger,
			)

			out, err := svc.GetTodo(context.Background(), todoquery.GetTodoQuery{ID: 1})
			if tt.wantKind != 0 {
				assertDomainKind(t, err, tt.wantKind)
				return
			}
			if err != nil {
				t.Fatalf("GetTodo() error = %v", err)
			}
			if out.ID != 1 {
				t.Fatalf("out.ID = %d, want 1", out.ID)
			}
		})
	}
}

func TestTodoService_CompleteTodoRunsInTransaction(t *testing.T) {
	now := time.Now()
	repo := &stubRepository{
		findItem: todo.UnmarshalFromDB(1, "write tests", todo.StatusPending, 1, now, now),
	}
	txRunner := &stubTxRunner{}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		txRunner,
		testLogger,
	)

	out, err := svc.CompleteTodo(context.Background(), todocmd.CompleteTodoCommand{ID: 1})
	if err != nil {
		t.Fatalf("CompleteTodo() error = %v", err)
	}

	if !txRunner.called {
		t.Fatal("txRunner.called = false, want complete use case to run in transaction")
	}
	if repo.saved == nil {
		t.Fatal("repository.saved = nil")
	}
	if out.Status != string(todo.StatusDone) {
		t.Fatalf("out.Status = %q, want %q", out.Status, todo.StatusDone)
	}
}

func TestTodoService_CompleteTodoReturnsNotFoundWhenSaveMisses(t *testing.T) {
	now := time.Now()
	repo := &stubRepository{
		findItem: todo.UnmarshalFromDB(1, "write tests", todo.StatusPending, 1, now, now),
		saveErr:  todo.ErrNotFound,
	}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		&stubTxRunner{},
		testLogger,
	)

	_, err := svc.CompleteTodo(context.Background(), todocmd.CompleteTodoCommand{ID: 1})
	assertNotFoundError(t, err)
}

func TestTodoService_CompleteTodoFailureBranches(t *testing.T) {
	now := time.Now()
	doneTodo := todo.UnmarshalFromDB(1, "write tests", todo.StatusDone, 1, now, now)
	pendingTodo := todo.UnmarshalFromDB(1, "write tests", todo.StatusPending, 1, now, now)
	tests := []struct {
		name     string
		repo     *stubRepository
		wantKind apperr.Kind
	}{
		{name: "find not found", repo: &stubRepository{}, wantKind: apperr.KindNotFound},
		{name: "find infra error", repo: &stubRepository{findErr: stderrors.New("select failed")}, wantKind: apperr.KindInternal},
		{name: "already done", repo: &stubRepository{findItem: doneTodo}},
		{name: "save infra error", repo: &stubRepository{findItem: pendingTodo, saveErr: stderrors.New("update failed")}, wantKind: apperr.KindInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewTodoService(
				tt.repo,
				&stubIDGenerator{},
				&stubTxRunner{},
				testLogger,
			)

			_, err := svc.CompleteTodo(context.Background(), todocmd.CompleteTodoCommand{ID: 1})
			if err == nil {
				t.Fatalf("CompleteTodo() error = nil, want error")
			}
			if tt.wantKind != 0 {
				assertDomainKind(t, err, tt.wantKind)
			}
		})
	}
}

func TestTodoService_DeleteTodoRunsInTransaction(t *testing.T) {
	repo := &stubRepository{}
	txRunner := &stubTxRunner{}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		txRunner,
		testLogger,
	)

	if err := svc.DeleteTodo(context.Background(), todocmd.DeleteTodoCommand{ID: 1}); err != nil {
		t.Fatalf("DeleteTodo() error = %v", err)
	}

	if !txRunner.called {
		t.Fatal("txRunner.called = false, want delete use case to run in transaction")
	}
	if repo.deleted != 1 {
		t.Fatalf("repository.deleted = %d, want 1", repo.deleted)
	}
}

func TestTodoService_DeleteTodoReturnsNotFoundWhenDeleteMisses(t *testing.T) {
	repo := &stubRepository{deleteErr: todo.ErrNotFound}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		&stubTxRunner{},
		testLogger,
	)

	err := svc.DeleteTodo(context.Background(), todocmd.DeleteTodoCommand{ID: 404})
	assertNotFoundError(t, err)
}

func TestTodoService_DeleteTodoWrapsInfrastructureError(t *testing.T) {
	repo := &stubRepository{deleteErr: stderrors.New("delete failed")}
	svc := NewTodoService(
		repo,
		&stubIDGenerator{},
		&stubTxRunner{},
		testLogger,
	)

	err := svc.DeleteTodo(context.Background(), todocmd.DeleteTodoCommand{ID: 1})
	assertDomainKind(t, err, apperr.KindInternal)
}

func assertNotFoundError(t *testing.T, err error) {
	t.Helper()

	var domainErr *apperr.DomainError
	if !stderrors.As(err, &domainErr) {
		t.Fatalf("error = %v, want DomainError", err)
	}
	if domainErr.Kind != apperr.KindNotFound {
		t.Fatalf("DomainError.Kind = %v, want %v", domainErr.Kind, apperr.KindNotFound)
	}
}

func assertDomainKind(t *testing.T, err error, want apperr.Kind) {
	t.Helper()

	var domainErr *apperr.DomainError
	if !stderrors.As(err, &domainErr) {
		t.Fatalf("error = %v, want DomainError", err)
	}
	if domainErr.Kind != want {
		t.Fatalf("DomainError.Kind = %v, want %v", domainErr.Kind, want)
	}
}
