package datainit

import (
	"context"
	"errors"
	"fmt"

	"github.com/maguowei/gotobeta/internal/app/bootstrap"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/localid"
	workspacepersist "github.com/maguowei/gotobeta/internal/modules/workspace/infra/persistence"
	workspaceseed "github.com/maguowei/gotobeta/internal/modules/workspace/infra/seed"
)

// Run 执行数据初始化。
//
// 依赖由调用方通过 *bootstrap.Runtime 注入；本函数负责装配数据库客户端
// 并执行初始化逻辑，可被 ctx 取消中断。
func Run(ctx context.Context, rt *bootstrap.Runtime) (err error) {
	client, sqlDB, err := entdb.NewEntClient(&rt.Cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close ent client: %w", closeErr))
		}
	}()
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close sql db: %w", closeErr))
		}
	}()

	// 初始化平台级 RBAC 模板（权限目录 + 模板角色，workspace_id=0），幂等可重入。
	rbacRepo := workspacepersist.NewRBACRepository(client, rt.AppLogger)
	if err := workspaceseed.SeedPlatformTemplates(ctx, rbacRepo, localid.New()); err != nil {
		return fmt.Errorf("seed platform rbac templates: %w", err)
	}
	rt.AppLogger.InfoContext(ctx, "platform rbac templates seeded")

	return nil
}
