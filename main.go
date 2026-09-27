package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/repository"
	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/config"
	"github.com/Alan-00280/go-pgsql-mhs.git/database"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/Alan-00280/go-pgsql-mhs.git/route"
)

const minJwtSecretLength = 32

func main() {
	// Load ENV
	config.LoadEnv()

	// Logger Config
	logger := config.NewLogger()

	// JWT Secret
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minJwtSecretLength {
		logger.Error("jwt secret isn't valid please check again", slog.Int("min length:", minJwtSecretLength))
		os.Exit(1)
	}

	// Create DB Pool
	pool, err := database.NewPool(context.Background())
	if err != nil {
		fmt.Printf("Err: %v", err)
		return
	}
	defer pool.Close()

	// JWT Manager
	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "be-prak"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// Load Role Permissions
	roleRepo := repository.NewRoleRepository(pool)
	rawRolePerm, err := roleRepo.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat role permission", slog.String("error", err.Error()))
		os.Exit(1)
	}

	permissionSet := helper.NewPermissionSet(rawRolePerm)
	logger.Info("berhasil memuat role permission", slog.Any("roles", permissionSet.KnownRoles()))

	// Load Common Password
	passwordCommonPath, err := filepath.Abs("./files/common_password.txt")
	if err != nil {
		logger.Error("gagal memuat password umum", slog.String("error", err.Error()))
		os.Exit(1)
	}

	passwordCommonSet, err := helper.NewPasswordCommonSet(passwordCommonPath)
	if err != nil {
		logger.Error("gagal memuat password umum", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// App Validator
	appValidator := helper.NewValidator(passwordCommonSet)

	// Repo -> Services
	userRepo := repository.NewUserRepository(pool)
	userService := service.NewUserHandler(userRepo, permissionSet, appValidator)

	authRepo := repository.NewAuthRepo(pool)
	authService := service.NewAuthHandler(
		userRepo,
		authRepo,
		jwtManager,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
		permissionSet,
		appValidator,
	)

	studentRepo := repository.NewStudentRepository(pool)
	studentService := service.NewStudentHandler(studentRepo, permissionSet)

	deps := route.Dependencies{
		Pool:           pool,
		JWT:            jwtManager,
		UserHandler:    userService,
		AuthHandler:    authService,
		StudentHandler: studentService,
		Permission:     permissionSet,
	}

	// APP
	app := config.NewApp(logger, deps)
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
