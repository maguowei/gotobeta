package authz

import (
	"context"
	"testing"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
)

// fakeResolverRepo 仅实现 CachedResolver 依赖的两个方法，其余由内嵌 nil 接口占位。
type fakeResolverRepo struct {
	rbac.Repository
	version      int64
	resolveCalls int
	actions      map[string]struct{}
}

func (f *fakeResolverRepo) GetUserVersion(context.Context, int64, int64) (int64, error) {
	return f.version, nil
}

func (f *fakeResolverRepo) ResolveUserActions(context.Context, int64, int64) (map[string]struct{}, error) {
	f.resolveCalls++
	return f.actions, nil
}

// mapKV 是进程内 KV 桩。
type mapKV struct{ m map[string]string }

func newMapKV() *mapKV { return &mapKV{m: map[string]string{}} }

func (k *mapKV) Get(_ context.Context, key string) (string, bool, error) {
	v, ok := k.m[key]
	return v, ok, nil
}

func (k *mapKV) Set(_ context.Context, key, value string, _ time.Duration) error {
	k.m[key] = value
	return nil
}

func TestCachedResolverHitsCacheOnSameVersion(t *testing.T) {
	repo := &fakeResolverRepo{version: 1, actions: map[string]struct{}{"message.send": {}}}
	resolver := NewCachedResolver(repo, newMapKV(), time.Minute)
	ctx := context.Background()

	first, err := resolver.ResolveUserActions(ctx, 1, 9)
	if err != nil || len(first) != 1 {
		t.Fatalf("first resolve = %v, err=%v", first, err)
	}
	second, err := resolver.ResolveUserActions(ctx, 1, 9)
	if err != nil || len(second) != 1 {
		t.Fatalf("second resolve = %v, err=%v", second, err)
	}
	if repo.resolveCalls != 1 {
		t.Fatalf("DB resolve calls = %d, want 1 (第二次应命中缓存)", repo.resolveCalls)
	}
}

func TestCachedResolverRefetchesAfterVersionBump(t *testing.T) {
	repo := &fakeResolverRepo{version: 1, actions: map[string]struct{}{"a": {}}}
	resolver := NewCachedResolver(repo, newMapKV(), time.Minute)
	ctx := context.Background()

	_, _ = resolver.ResolveUserActions(ctx, 1, 9)
	repo.version = 2 // 模拟授权变更递增版本
	_, _ = resolver.ResolveUserActions(ctx, 1, 9)

	if repo.resolveCalls != 2 {
		t.Fatalf("DB resolve calls = %d, want 2 (版本变更后应重新查询)", repo.resolveCalls)
	}
}

func TestCachedResolverNilKVDirectDB(t *testing.T) {
	repo := &fakeResolverRepo{version: 1, actions: map[string]struct{}{"a": {}}}
	resolver := NewCachedResolver(repo, nil, time.Minute)

	_, _ = resolver.ResolveUserActions(context.Background(), 1, 9)
	_, _ = resolver.ResolveUserActions(context.Background(), 1, 9)
	if repo.resolveCalls != 2 {
		t.Fatalf("DB resolve calls = %d, want 2 (无缓存直查)", repo.resolveCalls)
	}
}
