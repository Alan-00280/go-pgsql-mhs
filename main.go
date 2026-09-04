package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/config"
	"github.com/Alan-00280/go-pgsql-mhs.git/database"
)

func main() {
	// Load ENV
	config.LoadEnv()

	// Logger Config
	logger := config.NewLogger()

	// Create DB Pool
	pool, err := database.NewPool(context.Background())
	if err != nil {
		fmt.Printf("Err: %v", err)
		return
	}
	defer pool.Close()

	// Repo -> Services
	studentRepo := repository.NewStudentRepository(pool)
	studentService := service.NewStudentHandler(studentRepo)

	// APP
	app := config.NewApp(logger, pool, studentService)
	port := config.GetEnv("APP_PORT", "3000")

	// run
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server stopped! ", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("server running...", slog.String("port", port))

	// gracefull shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server . . .")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("fail to shutdown system! ", slog.String("error", err.Error()))
	}

	logger.Info("server shutted down")
}
