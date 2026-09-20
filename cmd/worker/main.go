package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️  .env not found, using system env")
	}

	// 2. Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// 3. Setup context dengan cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("worker starting")

	// 4. Placeholder: heartbeat setiap 10 detik
	// Nanti diganti dengan River worker setup
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Debug("worker heartbeat")
			}
		}
	}()

	// 5. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down worker...")
	cancel()

	// Beri waktu untuk job yang sedang berjalan
	time.Sleep(2 * time.Second)

	logger.Info("worker stopped")
}
