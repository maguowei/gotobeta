package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/maguowei/gotobeta/internal/app/bootstrap"
	"github.com/maguowei/gotobeta/internal/infra/metrics"
	systemrouter "github.com/maguowei/gotobeta/internal/modules/system/adapter/http/router"
	"github.com/maguowei/gotobeta/internal/pkg/health"
	httpmiddleware "github.com/maguowei/gotobeta/internal/pkg/httpx/middleware"

	"github.com/maguowei/gotobeta/internal/infra/cache"
	"github.com/maguowei/gotobeta/internal/infra/entdb"
	"github.com/maguowei/gotobeta/internal/infra/eventbus"
	"github.com/maguowei/gotobeta/internal/infra/objstore"

	"github.com/maguowei/gotobeta/internal/modules/user"

	"github.com/maguowei/gotobeta/internal/modules/media"
	"github.com/maguowei/gotobeta/internal/modules/messaging"
	"github.com/maguowei/gotobeta/internal/modules/realtime"
	"github.com/maguowei/gotobeta/internal/modules/todo"
	"github.com/maguowei/gotobeta/internal/modules/workspace"
)

var (
	listenAndServeHTTP = func(server *http.Server) error {
		return server.ListenAndServe()
	}
	shutdownHTTP = func(server *http.Server, ctx context.Context) error {
		return server.Shutdown(ctx)
	}

	newEntClient = entdb.NewEntClient
)

// RunHTTP 启动 HTTP 服务。
//
// 依赖由调用方通过 *bootstrap.Runtime 注入；本函数只负责装配中间件、
// 注册路由、监听端口并响应 ctx 取消（来自 main 的 signal handler）。
func RunHTTP(ctx context.Context, rt *bootstrap.Runtime) (err error) {
	cfg := rt.Cfg
	appLogger := rt.AppLogger
	auditLogger := rt.AuditLogger

	mc := metrics.NewCollectors(rt.MetricsRegistry, cfg.Metrics.Namespace)

	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(httpmiddleware.Recovery(appLogger))
	router.Use(httpmiddleware.BodyLimit(cfg.Server.MaxRequestBodyBytes))
	router.Use(httpmiddleware.CORS(cfg.Server.CORSAllowedOrigins))
	router.Use(httpmiddleware.TraceContext(cfg.Tracing.ServiceName))
	router.Use(httpmiddleware.RequestID())
	router.Use(httpmiddleware.Logger(appLogger))
	router.Use(httpmiddleware.Sentry())
	router.Use(httpmiddleware.Metrics(httpmiddleware.HTTPMetrics{
		RequestsTotal:   mc.HTTPRequestsTotal,
		RequestDuration: mc.HTTPRequestDuration,
	}))
	router.Use(httpmiddleware.Audit(auditLogger, httpmiddleware.AuditOptions{
		Enabled:             cfg.Audit.Enabled,
		LogRequestBody:      cfg.Audit.LogRequestBody,
		LogResponseBody:     cfg.Audit.LogResponseBody,
		MaskSensitiveFields: cfg.Audit.MaskSensitiveFields,
		MaxBodyBytes:        cfg.Audit.MaxBodyBytes,
	}))

	healthRegistry := health.NewRegistry()
	systemrouter.RegisterRoutes(router, healthRegistry)

	if cfg.Metrics.Enabled {
		router.GET(cfg.Metrics.Path, gin.WrapH(promhttp.HandlerFor(rt.MetricsRegistry, promhttp.HandlerOpts{})))
	}
	client, sqlDB, err := newEntClient(&cfg.Database)
	if err != nil {
		return err
	}
	defer func() {
		if client != nil {
			if closeErr := client.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close ent client: %w", closeErr))
			}
		}
	}()
	defer func() {
		if sqlDB != nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close sql db: %w", closeErr))
			}
		}
	}()

	// 注册 DB 就绪探针，独立于 demo/auth 业务模块
	healthRegistry.Register("database", health.CheckerFunc(func(ctx context.Context) error {
		return sqlDB.PingContext(ctx)
	}))

	redisClient, err := cache.NewRedisClient(cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if redisClient != nil {
			if closeErr := cache.CloseRedis(redisClient); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close redis client: %w", closeErr))
			}
		}
	}()
	redisKV := cache.NewRedisKV(redisClient)
	// readyz 探活 Redis（启用时）：依赖不可用时返回错误，避免流量打到坏实例。
	if redisClient != nil {
		healthRegistry.Register("redis", cache.RedisHealthChecker(redisClient))
	}

	apiV1 := router.Group("/api/v1")
	userMod, err := user.New(client, appLogger, cfg)
	if err != nil {
		return err
	}
	userMod.Mount(apiV1)
	todoMod, err := todo.New(client, appLogger, cfg)
	if err != nil {
		return err
	}
	// 启用 user-auth 时，demo 业务路由必须要求登录，避免在鉴权服务里出现公开可写端点。
	todoMod.Mount(apiV1, userMod.AuthMiddleware())

	workspaceMod, err := workspace.New(client, appLogger, cfg, redisKV)
	if err != nil {
		return err
	}
	workspaceMod.Mount(apiV1, userMod.AuthMiddleware())

	eventBus := eventbus.NewInProc(appLogger)
	messagingMod, err := messaging.New(client, appLogger, cfg, workspaceMod.Checker(), eventBus, mc)
	if err != nil {
		return err
	}
	messagingMod.Mount(apiV1, userMod.AuthMiddleware())

	realtimeMod, err := realtime.New(cfg, redisKV, messagingMod.MemberLookup(), messagingMod.ReadReporter(), eventBus, appLogger, mc)
	if err != nil {
		return err
	}
	realtimeMod.Mount(apiV1, userMod.AuthMiddleware())

	presigner, err := objstore.NewMinioPresigner(cfg.ObjStore)
	if err != nil {
		return err
	}
	// readyz 探活对象存储（配置了 endpoint 时）：bucket 不可达时返回错误。
	if presigner != nil {
		healthRegistry.Register("objstore", health.CheckerFunc(func(ctx context.Context) error {
			return presigner.CheckHealth(ctx)
		}))
	}
	mediaMod, err := media.New(client, appLogger, cfg, presigner, workspaceMod.Checker())
	if err != nil {
		return err
	}
	mediaMod.Mount(apiV1, userMod.AuthMiddleware())

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	if cfg.Server.MaxHeaderBytes > 0 {
		server.MaxHeaderBytes = cfg.Server.MaxHeaderBytes
	}

	errCh := make(chan error, 1)
	go func() {
		if err := listenAndServeHTTP(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// 先优雅关闭 WS：广播 close 帧并等待连接排空，客户端据此重连补拉，避免 HTTP 关闭时硬断。
		if err := realtimeMod.Shutdown(shutdownCtx); err != nil {
			appLogger.WarnContext(ctx, "realtime 优雅关闭未完全排空", slog.String("error", err.Error()))
		}
		if err := shutdownHTTP(server, shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return <-errCh
	}
}

// RunMigrate 执行数据库迁移。
func RunMigrate(ctx context.Context, rt *bootstrap.Runtime) error {
	return runMigrate(ctx, rt.Cfg, rt.AppLogger)
}
