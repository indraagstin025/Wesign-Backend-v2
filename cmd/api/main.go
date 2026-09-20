package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
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

	// 3. Setup Fiber
	app := fiber.New(fiber.Config{
		AppName:      "WeSign API v2.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// 4. Middleware dasar
	app.Use(requestIDMiddleware())
	app.Use(loggerMiddleware(logger))
	app.Use(recoveryMiddleware())

	// 5. Health check
	app.Get("/health", healthCheck)
	app.Get("/ready", readinessCheck)

	// 6. Placeholder API group
	api := app.Group("/api/v1")
	api.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "pong"})
	})

	// 7. Start server di goroutine
	port := getEnv("APP_PORT", "8080")

	go func() {
		logger.Info("api server starting", "port", port)
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// 8. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := app.ShutdownWithContext(ctx); err != nil {
		cancel()
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	cancel()

	logger.Info("server stopped")
}

// ============================================
// Middleware
// ============================================

func requestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get("X-Request-ID")
		if reqID == "" {
			reqID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Set("X-Request-ID", reqID)
		c.Locals("request_id", reqID)
		return c.Next()
	}
}

func loggerMiddleware(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		logger.Info("http request",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", duration.Milliseconds(),
			"request_id", c.Locals("request_id"),
		)
		return err
	}
}

func recoveryMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "error", r, "path", c.Path())
				if err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fiber.Map{
						"code":    "INTERNAL_ERROR",
						"message": "Internal server error",
					},
				}); err != nil {
					slog.Error("failed to write error response", "error", err)
				}
			}
		}()
		return c.Next()
	}
}

// ============================================
// Handlers
// ============================================

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func readinessCheck(c *fiber.Ctx) error {
	// TODO: cek koneksi DB, Redis, dll.
	return c.JSON(fiber.Map{
		"status": "ready",
	})
}

// ============================================
// Helpers
// ============================================

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
