package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
)

// KV 是缓存依赖的最小键值接口（由 infra/cache.RedisKV 实现）。
type KV interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

// CachedResolver 包装 rbac.Repository，为 ResolveUserActions 提供版本化缓存。
//
// 缓存键 perm:user:{ws}:{uid}:v{version}，version 由授权变更递增（见 BumpUserVersion），
// 因此权限变更后旧键自然失效（精准失效，不依赖 TTL 过期）。其余方法透传底层仓储。
type CachedResolver struct {
	rbac.Repository
	kv  KV
	ttl time.Duration
}

// NewCachedResolver 创建带缓存的解析器。kv 为 nil 时退化为直查底层仓储。
func NewCachedResolver(repo rbac.Repository, kv KV, ttl time.Duration) *CachedResolver {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &CachedResolver{Repository: repo, kv: kv, ttl: ttl}
}

// ResolveUserActions 优先读版本化缓存，未命中查底层仓储并回填。
func (c *CachedResolver) ResolveUserActions(ctx context.Context, workspaceID, userID int64) (map[string]struct{}, error) {
	if c.kv == nil {
		return c.Repository.ResolveUserActions(ctx, workspaceID, userID)
	}

	version, err := c.Repository.GetUserVersion(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("perm:user:%d:%d:v%d", workspaceID, userID, version)

	if raw, found, err := c.kv.Get(ctx, key); err == nil && found {
		var codes []string
		if json.Unmarshal([]byte(raw), &codes) == nil {
			set := make(map[string]struct{}, len(codes))
			for _, code := range codes {
				set[code] = struct{}{}
			}
			return set, nil
		}
	}

	actions, err := c.Repository.ResolveUserActions(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(actions))
	for code := range actions {
		codes = append(codes, code)
	}
	if encoded, err := json.Marshal(codes); err == nil {
		_ = c.kv.Set(ctx, key, string(encoded), c.ttl)
	}
	return actions, nil
}
