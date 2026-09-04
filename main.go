package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/config"
	"github.com/Alan-00280/go-pgsql-mhs.git/database"
	"github.com/Alan-00280/go-pgsql-mhs.git/middleware"
	"github.com/gofiber/fiber/v2"
)

// var bodiedMethod = map[string]bool{
// 	fiber.MethodPost:  true,
// 	fiber.MethodPut:   true,
// 	fiber.MethodPatch: true,
// }

// func requireJSON(c *fiber.Ctx) error {
// 	if bodiedMethod[c.Method()] {
// 		ct := c.Get("Content-Type")
// 		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
// 			return fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
// 		}
// 	}
// 	return c.Next()
// }

func main() {
	// Load ENV
	config.LoadEnv()

	// Create DB Pool
	pool, err := database.NewPool(context.Background())
	if err != nil {
		fmt.Printf("Err: %v", err)
		return
	}
	defer pool.Close()

	app := fiber.New(fiber.Config{
		AppName: "api-students",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			message := "terjadi kesalahan pada server"

			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				message = e.Message
			}

			return fail(c, status, message)
		},
	})

	// Global Middleware
	logHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	middleware.Register(app, slog.New(logHandler))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	// App Handlers
	student := api.Group("/students", middleware.RequireJSON)
	studentRepo := repository.NewStudentRepository(pool)
	studentHandler := service.NewStudentHandler(studentRepo)

	student.Get("/", studentHandler.List)
	student.Get("/:id", studentHandler.Get)
	student.Post("/", studentHandler.Create)
	student.Put("/:id", studentHandler.Replace)
	student.Patch("/:id", studentHandler.Patch)
	student.Delete("/:id", studentHandler.Delete)

	// 404
	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "404 - Route not found")
	})

	// Run
	port := "3000"
	fmt.Printf("Server running in %s... \n", port)
	log.Fatal(app.Listen("localhost:" + port))
}
