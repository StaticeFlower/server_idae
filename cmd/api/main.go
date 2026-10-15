package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"ideaspark-api/internal/auth"
	"ideaspark-api/internal/config"
	"ideaspark-api/internal/database"
	"ideaspark-api/internal/handlers"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// โหลด .env ถ้ามี (บน production ใช้ environment variables จริงแทน)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	if cfg.AutoMigrate {
		slog.Info("running AutoMigrate")
		if err := database.Migrate(db); err != nil {
			return err
		}
	}

	h := &handlers.Handler{
		DB:   db,
		Auth: auth.NewManager(cfg.JWTSecret, cfg.JWTTTL),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.Routes(cfg.CORSOrigins),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "cors_origins", cfg.CORSOrigins)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}
	return nil
}
