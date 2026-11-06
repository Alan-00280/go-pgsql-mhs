package route

import (
	"context"
	"time"

	"github.com/Alan-00280/go-pgsql-mhs.git/app/service"
	"github.com/Alan-00280/go-pgsql-mhs.git/helper"
	"github.com/Alan-00280/go-pgsql-mhs.git/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Dependencies struct {
	Pool              *pgxpool.Pool
	Mongoclient       *mongo.Client
	JWT               *helper.JWTManager
	Permission        *helper.PermissionSet
	StudentHandler    *service.StudentHandler
	AuthHandler       *service.AuthHandler
	UserHandler       *service.UserHandler
	CourseHandler     *service.CourseService
	SupervisorHandler *service.SupervisorServices
	ReviewHandler     *service.ReviewService
	PortofolioHandler *service.PortofolioService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	permissions := deps.Permission

	// PUBLIK
	api.Get("/health", healthCheck(deps.Pool, deps.Mongoclient))

	// AUTHENTICATION
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthHandler.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthHandler.Login)
	auth.Post("/refresh", deps.AuthHandler.Refresh)
	auth.Post("/logout", deps.AuthHandler.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthHandler.Me)

	// PROTECTED
	user := api.Group("/users", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	user.Get("/", middleware.RequirePermission(permissions, "user:list"), deps.UserHandler.ListAll)
	user.Post("/", middleware.RequirePermission(permissions, "user:update:any"), deps.UserHandler.Create)
	user.Delete("/:id", middleware.RequirePermission(permissions, "user:delete"), deps.UserHandler.Delete)
	user.Patch("/:id/role", middleware.RequirePermission(permissions, "role:assign"), deps.UserHandler.AssignRole)

	user.Put("/:id", deps.UserHandler.Replace)
	user.Get("/:id", deps.UserHandler.Get)
	user.Patch("/:id", deps.UserHandler.Patch)

	student := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	student.Get("/", middleware.RequirePermission(permissions, "student:list"), deps.StudentHandler.List)
	student.Get("/c", middleware.RequirePermission(permissions, "student:list"), deps.StudentHandler.ListCursor)
	student.Post("/", middleware.RequirePermission(permissions, "student:create"), deps.StudentHandler.Create)
	student.Delete("/:id", middleware.RequirePermission(permissions, "student:delete"), deps.StudentHandler.Delete)

	student.Put("/:id", deps.StudentHandler.Replace)
	student.Patch("/:id", deps.StudentHandler.Patch)
	student.Get("/:id", deps.StudentHandler.Get)

	student.Get("/:id/portfolio", deps.PortofolioHandler.Get)
	student.Put("/:id/portfolio", deps.PortofolioHandler.Put)
	student.Post("/:id/portfolio/projects", deps.PortofolioHandler.AddProject)
	student.Delete("/:id/portfolio/projects/:projectId", deps.PortofolioHandler.DeleteProject)

	// --- kelas dan KRS ---
	courses := api.Group("/courses", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))

	// "/mine" didaftarkan SEBELUM "/:id". Fiber mencocokkan route sesuai
	// urutan pendaftaran; bila dibalik, "mine" akan dianggap sebagai :id.
	courses.Get("/", deps.CourseHandler.List)
	courses.Get("/mine", deps.CourseHandler.Mine)
	courses.Get("/:id", deps.CourseHandler.Get)
	courses.Patch("/:id", middleware.RequirePermission(permissions, "course:manage"), deps.CourseHandler.UpdateQuota)

	courses.Post("/:id/enroll", deps.CourseHandler.Enroll)
	courses.Delete("/:id/enroll", deps.CourseHandler.Drop)

	courses.Get("/:id/reviews", deps.ReviewHandler.ListByCourse)
	courses.Get("/:id/reviews/summary", deps.ReviewHandler.Summary)
	courses.Post("/:id/reviews", deps.ReviewHandler.Create)

	enrollemnts := api.Group("/enrollments", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	enrollemnts.Post("/swap", deps.CourseHandler.Swap)

	// supervisors
	supervisors := api.Group("/supervisors", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))

	supervisors.Get("/", deps.SupervisorHandler.List)
	supervisors.Post("/mine", deps.SupervisorHandler.Mine)
	supervisors.Get("/:id", deps.SupervisorHandler.Get)

	supervisors.Post("/:id/assign", middleware.RequirePermission(deps.Permission, "supervisions:manage"), deps.SupervisorHandler.Assign)
	supervisors.Post("/switch", deps.SupervisorHandler.Switch)
	supervisors.Patch("/:id", deps.SupervisorHandler.UpdateQuota)

	// review
	reviews := api.Group("/reviews", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	reviews.Get("/:id", deps.ReviewHandler.Get)
	reviews.Patch("/:id", deps.ReviewHandler.Update)
	reviews.Delete("/:id", deps.ReviewHandler.Delete)
	reviews.Post("/:id/replies", deps.ReviewHandler.Reply)
	reviews.Post("/:id/helpful", deps.ReviewHandler.MarkHelpful)

	// portofolio
	portofolios := api.Group("/portfolios", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
	portofolios.Get("/stats/technologies", deps.PortofolioHandler.TechSummary)
}

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			// ERR 503 - Service Unavailable
			return helper.ServiceUnavailable("PSQL database can't be reached")
		}

		if err := mongoClient.Ping(ctx, nil); err != nil {
			// ERR 503 - Service Unavailable
			return helper.ServiceUnavailable("MongoDB database can't be reached")
		}

		return helper.Ok(c, "server and databases is OK!", nil)
	}
}
