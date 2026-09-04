package config

import (
	"log/slog"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/Alan-00280/go-pgsql-mhs.git/middleware"
	"github.com/Alan-00280/go-pgsql-mhs.git/route"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "terjadi kesalahan pada server"

		if e, ok := err.(*fiber.Error); ok {
			status = e.Code
			message = e.Message
		}

		logger.Error(
			"unhandled_error",
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("error", err.Error()),
		)

		return helper.Fail(c, status, message)
	}
}

func NewApp(logger *slog.Logger, pool *pgxpool.Pool, studentService *service.StudentHandler) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "api-backend"),
		ErrorHandler: newErrorHandler(logger),
	})

	// Middleware
	middleware.Register(app, logger)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// App Routes
	route.Register(app, pool, studentService)

	// 404
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "404 - Route not found")
	})

	return app
}
