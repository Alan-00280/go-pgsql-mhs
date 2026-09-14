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

type Dependencies struct {
	Pool           *pgxpool.Pool
	JWT            *helper.JWTManager
	StudentHandler *service.StudentHandler
	AuthHandler    *service.AuthHandler
	UserHandler    *service.UserHandler
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// PUBLIK
	api.Get("/health", healthCheck(deps.Pool))

	// AUTHENTICATION
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthHandler.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthHandler.Login)
	auth.Post("/refresh", deps.AuthHandler.Refresh)
	auth.Post("/logout", deps.AuthHandler.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthHandler.Me)

	// PROTECTED
	user := api.Group("/users", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	user.Get("/", deps.UserHandler.ListAll)
	user.Get("/:id", deps.UserHandler.Get)
	user.Post("/", deps.UserHandler.Create)
	user.Patch("/:id", deps.UserHandler.Patch)
	user.Delete("/:id", deps.UserHandler.Delete)

	student := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	student.Get("/", deps.StudentHandler.List)
	student.Get("/:id", deps.StudentHandler.Get)
	student.Post("/", deps.StudentHandler.Create)
	student.Put("/:id", deps.StudentHandler.Replace)
	student.Patch("/:id", deps.StudentHandler.Patch)
	student.Delete("/:id", deps.StudentHandler.Delete)
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
