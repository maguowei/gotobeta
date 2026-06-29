package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/maguowei/gotobeta/internal/app/bootstrap"
	"github.com/maguowei/gotobeta/internal/app/server"
)

var (
	bootstrapInit = bootstrap.Init
	runMigrate    = server.RunMigrate
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	// 长时间迁移需要可被 SIGINT/SIGTERM 中断，避免运维只能 kill -9 引发不一致。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rt, err := bootstrapInit(ctx, bootstrap.Options{
		ProcessName:  "migrate",
		EnableTracer: false, // 短命进程不开 trace，避免 OTLP 拨号拖慢冷启动
	})
	if err != nil {
		return err
	}
	defer func() {
		if shutdownErr := rt.Shutdown(context.Background()); shutdownErr != nil {
			err = errors.Join(err, shutdownErr)
		}
	}()

	if err := runMigrate(ctx, rt); err != nil {
		rt.AppLogger.ErrorContext(ctx, "migration failed", slog.Any("error", err))
		return err
	}

	return nil
}
