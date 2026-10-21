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

	"webhookhub/internal/config"
	"webhookhub/internal/event"
	"webhookhub/internal/httpapi"
	"webhookhub/internal/storage/memory"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()
	events := event.NewService(memory.NewEventStore())
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewHandler(events),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server starting", "address", cfg.HTTPAddr)
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		logger.Info("HTTP server shutting down")
	}
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return err
	}
	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
