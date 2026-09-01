package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

var bodiedMethod = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

func requireJSON(c *fiber.Ctx) error {
	if bodiedMethod[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
		}
	}
	return c.Next()
}

func main() {
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
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${method} ${path} ${status} ${latency}\n",
	}))
	app.Use(cors.New())

	// API v1 //
	api := app.Group("/api/v1")
	api.Get("/", func(c *fiber.Ctx) error {
		return ok(c, "Health Check - Server OK!", fiber.Map{"timestamp": time.Now()})
	})

	// App Handlers
	student := api.Group("/students", requireJSON)
	student.Get("/", listStudents)
	student.Get("/:id", getStudent)
	student.Post("/", createStudent)
	student.Put("/:id", replaceStudent)
	student.Patch("/:id", patchStudent)
	student.Delete("/:id", deleteStudent)

	// 404
	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "404 - Route not found")
	})

	// Run
	port := "3000"
	fmt.Printf("Server running in %s... \n", port)
	log.Fatal(app.Listen("127.0.0.1:" + port))
}
