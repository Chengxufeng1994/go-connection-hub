package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"
	"go-notification-sse/internal/infrastructure/server"
)

func main() {
	ctx := context.Background()
	sctx := WithSignal(ctx)

	lCfg := logger.NewDefaultConfig()
	log := logger.NewLogrusLogger(lCfg)
	hubInstance := hub.New(log)

	// Start the hub first
	if err := hubInstance.Start(ctx); err != nil {
		log.Errorf("failed to start hub: %v", err)
		return
	}
	log.Infof(
		"hub started before router initialization, running status: %v",
		hubInstance.IsRunning(),
	)

	router := InitRouter(hubInstance, log)
	httpSrv := server.NewHTTPServer(router)
	app := newApplication(log, httpSrv, hubInstance)
	if err := app.Run(sctx); err != nil {
		log.Errorf("failed to run application: %v", err)
	}
}

type Application struct {
	logger  logger.Logger
	httpSrv server.Server
	hub     *hub.Hub
}

func newApplication(
	logger logger.Logger,
	httpSrv *server.HTTPServer,
	hubInstance *hub.Hub,
) *Application {
	return &Application{
		logger:  logger.WithField("app", "sse"),
		httpSrv: httpSrv,
		hub:     hubInstance,
	}
}

func (app *Application) Run(ctx context.Context) error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		return app.httpSrv.Start(ctx)
	})

	eg.Go(func() error {
		<-ctx.Done()

		gracefulshutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(5*time.Second),
		)
		defer cancel()

		// Stop hub first
		if err := app.hub.Stop(gracefulshutdownCtx); err != nil {
			app.logger.Errorf("failed to stop hub: %v", err)
		}

		return app.httpSrv.Stop(gracefulshutdownCtx)
	})

	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

func WithSignal(pctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(pctx)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

		<-sigc

		cancel()
	}()

	return ctx
}
