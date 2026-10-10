// Command iotlab-service starts the HTTP API and the worker group, and shuts both down on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "go.uber.org/automaxprocs"

	server "github.com/tdatIT/iotlab-service/internal"
)

// @title           IoT Service API
// @version         1.0.0
// @description     HTTP API of the iotlab-service. Every endpoint answers with the shared
// @description     response envelope: `code` carries the domain outcome, `data` the payload.
// @host            localhost:3000
// @BasePath        /api/v1
func main() {
	slog.Info("starting app")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serv, err := server.NewServer()
	if err != nil {
		slog.Error("error while creating server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	var wg sync.WaitGroup

	// HTTP component
	wg.Go(func() {
		if err := serv.API().Start(serv.Config().Server.Port); err != nil {
			slog.Error("http server stopped", slog.String("error", err.Error()))
		}
	})

	// Worker component
	wg.Go(func() {
		if err := serv.Workers().StartGroup(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("worker group stopped with error", slog.String("error", err.Error()))
		}
	})

	// Block until signal arrives, then start shutdown sequence.
	<-ctx.Done()
	stop() // release signal resources early

	slog.Info("gracefully shutting down")
	serv.Shutdown()

	wg.Wait()
	slog.Info("server shutdown successfully")
	os.Exit(0)
}
