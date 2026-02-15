package handler

import (
	"fmt"
	"time"

	"github.com/G0tem/go-service-task/internal/config"
	"github.com/G0tem/go-service-task/internal/model"
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/golang-jwt/jwt/v5"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gorm.io/gorm"
)

type Handler struct {
	db      *gorm.DB
	redis   *redis.Client
	cfg     *config.Config
	metrics *fiberprometheus.FiberPrometheus // Добавляем метрики
}

func NewHandler(db *gorm.DB, rds *redis.Client, cfg *config.Config) *Handler {
	return &Handler{
		db:      db,
		cfg:     cfg,
		redis:   rds,
		metrics: fiberprometheus.New("task-management-service"),
	}
}

func (h *Handler) SetupRoutes(app *fiber.App) {
	h.metrics.RegisterAt(app, "/metrics")
	app.Use(h.metrics.Middleware)

	// Rate limiter middleware (100 запросов/мин) из допов ТЗ.
	rateLimit := limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Извлекаем пользователя из JWT
			user := c.Locals("user")
			if user != nil {
				if u, ok := user.(*model.User); ok {
					return fmt.Sprintf("rate_limit:%d", u.ID)
				}
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "too many requests",
				"retry_after": 60,
			})
		},
	})

	api := app.Group("api")
	v1 := api.Group("v1")

	docs := v1.Group("docs")
	docs.Get("*", fiberSwagger.WrapHandler)

	auth := v1.Group("auth")
	auth.Use(rateLimit) // Для публичных эндпоинтов лимит по IP
	// Публичные маршруты - без проверки JWT
	auth.Post("login", h.login)
	auth.Post("register", h.register)

	// Защищенные маршруты с middleware JWT
	protected := v1.Group("/")
	protected.Use(JWTMiddleware(h.cfg.SecretKey))
	protected.Use(rateLimit)

	// Auth protected
	protected.Get("auth/get-me", h.getMe)
	protected.Post("auth/refresh", h.refresh)

	// Teams
	protected.Post("teams", h.createTeam)
	protected.Get("teams", h.listUserTeams)
	protected.Post("teams/:id/invite", h.inviteToTeam)

	// Tasks
	protected.Post("tasks", h.createTask)
	protected.Get("tasks", h.listTasks)
	protected.Put("tasks/:id", h.updateTask)
	protected.Get("tasks/:id/history", h.getTaskHistory)

	// System endpoints with complex SQL
	system := protected.Group("system")
	system.Get("teams-stats", h.getTeamStats)
	system.Get("teams-top-creators", h.getTopCreatorsByTeam)
	system.Get("invalid-assignees", h.getTasksWithInvalidAssignee)
}

func (h *Handler) GetJWT(user *model.User) (string, error) {
	// Create the Claims
	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"email":    user.Email,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(h.cfg.SecretKey))

	return t, err
}
