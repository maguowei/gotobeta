package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/maguowei/gotobeta/internal/app/bootstrap"
	"github.com/maguowei/gotobeta/internal/app/datainit"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	// 数据初始化可能很久，需要可被 SIGINT/SIGTERM 中断，避免 kill -9 留下脏数据。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rt, err := bootstrap.Init(ctx, bootstrap.Options{
		ProcessName:  "datainit",
		EnableTracer: false, // 短命进程不开 trace
	})
	if err != nil {
		return err
	}
	defer func() {
		if shutdownErr := rt.Shutdown(context.Background()); shutdownErr != nil {
			err = errors.Join(err, shutdownErr)
		}
	}()

	return datainit.Run(ctx, rt)
}
