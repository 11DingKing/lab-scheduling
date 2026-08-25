package app

import (
	"context"
	"errors"
	"github.com/11DingKing/lab-scheduling/internal/auth"
	"github.com/11DingKing/lab-scheduling/internal/clock"
	"github.com/11DingKing/lab-scheduling/internal/config"
	"github.com/11DingKing/lab-scheduling/internal/httpapi"
	"github.com/11DingKing/lab-scheduling/internal/migration"
	"github.com/11DingKing/lab-scheduling/internal/observability"
	"github.com/11DingKing/lab-scheduling/internal/repository"
	"github.com/11DingKing/lab-scheduling/internal/service"
	"github.com/11DingKing/lab-scheduling/internal/worker"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(ctx context.Context) error {
	cfg := config.Load()
	logger := observability.NewLogger()
	db, err := repository.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = migration.Apply(ctx, db.SQL, cfg.MigrationsPath); err != nil {
		return err
	}
	now := clock.Real{}
	authSvc := auth.Service{Store: db, Clock: now, TTL: cfg.SessionTTL}
	svc := service.Service{Store: db, Clock: now}
	q := worker.New(32, 3)
	q.Start(ctx)
	defer q.Stop()
	server := &httpapi.Server{Auth: authSvc, Service: svc, Queries: db}
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Info("lab scheduling started", map[string]any{"addr": cfg.HTTPAddr})
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
func Main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := Run(ctx); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
