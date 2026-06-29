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
	runHTTP       = server.RunHTTP
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rt, err := bootstrapInit(ctx, bootstrap.Options{
		ProcessName:  "server",
		EnableTracer: true,
	})
	if err != nil {
		return err
	}
	defer func() {
		if shutdownErr := rt.Shutdown(context.Background()); shutdownErr != nil {
			err = errors.Join(err, shutdownErr)
		}
	}()

	if err := runHTTP(ctx, rt); err != nil {
		rt.AppLogger.ErrorContext(ctx, "server stopped with error", slog.Any("error", err))
		return err
	}

	return nil
}
