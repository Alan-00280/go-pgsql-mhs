package main

import (
	"context"
	"flag"
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
	_ "github.com/Alan-00280/go-pgsql-mhs.git/docs" // hasil swag init; mendaftarkan spesifikasi ke swagger
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/Alan-00280/go-pgsql-mhs.git/route"
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

const minJwtSecretLength = 32

// Blok komentar di bawah ini adalah informasi UMUM dokumentasi API. swag
// hanya membacanya dari file yang disebut pada flag -g (bawaannya main.go).
//
// @title                       API KRS Praktikum Backend Lanjut
// @version                     1.0
// @description                 API pendaftaran kelas (KRS), ulasan kelas, dan notifikasi.
// @description                 Seluruh endpoint selain /auth memerlukan access token.
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Isi dengan: Bearer <access_token>
func main() {
	config.LoadEnv()
	logger := config.NewLogger()

	// Satu program, tiga cara menjalankan:
	//   go run . -role=api     hanya melayani HTTP
	//   go run . -role=worker  hanya relay outbox dan worker queue
	//   go run . -role=all     keduanya dalam satu proses
	// Di production, api dan worker dijalankan sebagai proses terpisah agar
	// masing-masing dapat diperbanyak sesuai bebannya.
	role := flag.String("role", config.GetEnv("APP_ROLE", "all"), "api | worker | all")
	flag.Parse()
	runAPI := *role == "api" || *role == "all"
	runWorker := *role == "worker" || *role == "all"
	if !runAPI && !runWorker {
		logger.Error("role tidak dikenal", slog.String("role", *role))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// ctx dibatalkan ketika Ctrl+C ditekan atau sistem mengirim SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var app *fiber.App
	if runAPI {
		app = startAPI(logger, pool)
		if app == nil {
			os.Exit(1)
		}
	}

	var stopWorker func()
	if runWorker {
		stopWorker = startWorker(ctx, logger, pool)
		if stopWorker == nil {
			os.Exit(1)
		}
	}

	logger.Info("aplikasi berjalan", slog.String("role", *role))
	<-ctx.Done()

	// Urutan berhenti: pintu masuk ditutup lebih dulu (HTTP), baru kemudian
	// pekerja di belakangnya.
	if app != nil {
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			logger.Error("shutdown HTTP tidak rapi", slog.String("error", err.Error()))
		}
	}
	if stopWorker != nil {
		stopWorker()
	}
	logger.Info("aplikasi berhenti dengan rapi")
}

// startAPI menyiapkan seluruh kebutuhan HTTP. Mengembalikan nil bila gagal.
func startAPI(logger *slog.Logger, pool *pgxpool.Pool) *fiber.App {
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minJwtSecretLength {
		logger.Error("jwt secret isn't valid please check again", slog.Int("min length:", minJwtSecretLength))
		return nil
	}

	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("[ERROR]", err.Error()))
		return nil
	}
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_be"))

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "be-prak"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	roleRepo := repository.NewRoleRepository(pool)
	rawRolePerm, err := roleRepo.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat role permission", slog.String("error", err.Error()))
		return nil
	}

	permissionSet := helper.NewPermissionSet(rawRolePerm)
	logger.Info("berhasil memuat role permission", slog.Any("roles", permissionSet.KnownRoles()))

	passwordCommonPath, err := filepath.Abs("./files/common_password.txt")
	if err != nil {
		logger.Error("gagal memuat password umum", slog.String("error", err.Error()))
		return nil
	}

	passwordCommonSet, err := helper.NewPasswordCommonSet(passwordCommonPath)
	if err != nil {
		logger.Error("gagal memuat password umum", slog.String("error", err.Error()))
		return nil
	}

	appValidator := helper.NewValidator(passwordCommonSet)
	txManager := repository.NewTxManager(pool)

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
	studentService := service.NewStudentHandler(studentRepo, permissionSet, appValidator)

	courseRepo := repository.NewCourseRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	courseService := service.NewCourseService(courseRepo, outboxRepo, txManager, appValidator)

	supervisorRepo := repository.NewSupervisorRepository(pool)
	superviserService := service.NewSupervisorServices(supervisorRepo, txManager, appValidator)

	reviewRepo := repository.NewReviewRepository(mongoDB)
	reviewService := service.NewReviewService(reviewRepo, courseRepo, permissionSet, appValidator)
	if err := reviewRepo.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index MongoDB", slog.String("[ERROR]", err.Error()))
	}

	portofolioRepo := repository.NewPortofolioRepository(mongoDB)
	portofolioService := service.NewPortofolioService(portofolioRepo, studentRepo, permissionSet, appValidator)
	if err := portofolioRepo.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index portofolio MongoDB", slog.String("[ERROR]", err.Error()))
	}

	inspector := asynq.NewInspector(config.RedisOpt())
	notificationRepo := repository.NewNotificationRepository(pool)
	notificationService := service.NewNotificationService(notificationRepo, outboxRepo, inspector)

	deps := route.Dependencies{
		Pool:                pool,
		Mongoclient:         mongoClient,
		JWT:                 jwtManager,
		Permission:          permissionSet,
		UserHandler:         userService,
		AuthHandler:         authService,
		StudentHandler:      studentService,
		CourseHandler:       courseService,
		SupervisorHandler:   superviserService,
		ReviewHandler:       reviewService,
		PortofolioHandler:   portofolioService,
		NotificationService: notificationService,
		EnableSwagger:       config.GetEnv("SWAGGER_ENABLED", "false") == "true",
	}

	app := config.NewApp(logger, deps)

	app.Hooks().OnShutdown(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = inspector.Close()
		return mongoClient.Disconnect(ctx)
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server stopped! ", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("server running...", slog.String("port", port))
	return app
}
