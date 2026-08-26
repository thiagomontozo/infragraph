package main

import (
	"context"
	"errors"
	"github.com/thiagomontozo/infragraph/internal/app"
	"github.com/thiagomontozo/infragraph/internal/config"
	"github.com/thiagomontozo/infragraph/internal/database"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, e := config.Load()
	if e != nil {
		slog.Error("configuration rejected", "error", e)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	db, e := database.Open(ctx, cfg.DatabaseURL)
	cancel()
	if e != nil {
		slog.Error("database unavailable", "error", e)
		os.Exit(1)
	}
	defer db.Close()
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	e = db.Migrate(ctx)
	cancel()
	if e != nil {
		slog.Error("migration failed", "error", e)
		os.Exit(1)
	}
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: app.New(cfg, db, slog.Default()).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 90 * time.Second, MaxHeaderBytes: 1 << 20}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		slog.Info("InfraGraph API listening", "address", cfg.HTTPAddr)
		if e := server.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "error", e)
			os.Exit(1)
		}
	}()
	<-stopped
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if e := server.Shutdown(ctx); e != nil {
		slog.Error("graceful shutdown exceeded deadline", "error", e)
	}
}
