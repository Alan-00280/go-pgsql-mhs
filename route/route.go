package route

import (
	"context"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/Alan-00280/go-pgsql-mhs.git/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(app *fiber.App, pool *pgxpool.Pool, studentHandler *service.StudentHandler) {
	api := app.Group("/api/v1")

	api.Get("/health", healthCheck(pool))

	student := api.Group("/students", middleware.RequireJSON)
	student.Get("/", studentHandler.List)
	student.Get("/:id", studentHandler.Get)
	student.Post("/", studentHandler.Create)
	student.Put("/:id", studentHandler.Replace)
	student.Patch("/:id", studentHandler.Patch)
	student.Delete("/:id", studentHandler.Delete)
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			// ERR 503 - Service Unavailable
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database can't be reached")
		}

		return helper.Ok(c, "server and database is OK!", nil)
	}
}
