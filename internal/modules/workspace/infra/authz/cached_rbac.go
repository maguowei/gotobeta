package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/maguowei/gotobeta/internal/modules/workspace/domain/rbac"
	loggerx "github.com/maguowei/gotobeta/internal/pkg/logger"
)

// permCacheTTL 是权限动作集缓存的兜底过期时间；精准失效靠版本号变更，TTL 仅防缓存泄漏。
const permCacheTTL = 5 * time.Minute

// PermCache 是权限缓存所需的最小 KV 能力（*cache.RedisKV 满足）。
// 定义在使用方以便依赖倒置与测试 fake。
type PermCache interface {
	Get(ctx context.Context, key string) (value string, found bool, err error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

// CachedRBAC 包装 rbac.Repository，对热路径 ResolveUserActions 增加版本化缓存：
// 缓存键 perm:user:{ws}:{uid}:v{version}，授权变更经 BumpPermissionVersion 递增版本即令旧缓存失效。
// cache 为 nil 时（Redis 关闭）透明退化为直查；其余方法走嵌入仓储。
type CachedRBAC struct {
	rbac.Repository
	cache  PermCache
	logger *slog.Logger
}

// NewCachedRBAC 创建带缓存的 RBAC 仓储。cache 为 nil 时所有读直查底层。
func NewCachedRBAC(repo rbac.Repository, cache PermCache, logger *slog.Logger) *CachedRBAC {
	return &CachedRBAC{Repository: repo, cache: cache, logger: logger}
}

// ResolveUserActions 优先读缓存，未命中查底层并回填。缓存读写失败均降级直查，不阻断鉴权。
func (c *CachedRBAC) ResolveUserActions(ctx context.Context, workspaceID, userID int64) (map[string]struct{}, error) {
	if c.cache == nil {
		return c.Repository.ResolveUserActions(ctx, workspaceID, userID)
	}

	version, err := c.PermissionVersion(ctx, workspaceID, userID)
	if err != nil {
		return c.Repository.ResolveUserActions(ctx, workspaceID, userID)
	}
	key := fmt.Sprintf("perm:user:%d:%d:v%d", workspaceID, userID, version)

	if raw, found, gerr := c.cache.Get(ctx, key); gerr == nil && found {
		var codes []string
		if json.Unmarshal([]byte(raw), &codes) == nil {
			return toActionSet(codes), nil
		}
	}

	set, err := c.Repository.ResolveUserActions(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if raw, merr := json.Marshal(actionCodes(set)); merr == nil {
		if serr := c.cache.Set(ctx, key, string(raw), permCacheTTL); serr != nil {
			loggerx.WithError(ctx, c.logger, "perm cache set failed", serr,
				slog.Int64("workspaceId", workspaceID), slog.Int64("userId", userID))
		}
	}
	return set, nil
}

func toActionSet(codes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		set[code] = struct{}{}
	}
	return set
}

func actionCodes(set map[string]struct{}) []string {
	codes := make([]string, 0, len(set))
	for code := range set {
		codes = append(codes, code)
	}
	return codes
}
