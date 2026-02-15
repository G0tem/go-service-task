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
	"github.com/rs/zerolog/log"
	"github.com/sony/gobreaker"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"gorm.io/gorm"
)

type Handler struct {
	db           *gorm.DB
	redis        *redis.Client
	cfg          *config.Config
	metrics      *fiberprometheus.FiberPrometheus // Добавляем метрики
	emailBreaker *gobreaker.CircuitBreaker        // Добавляем CircuitBreaker
}

func NewHandler(db *gorm.DB, rds *redis.Client, cfg *config.Config) *Handler {
	// Настраиваем Circuit Breaker для email сервиса
	settings := gobreaker.Settings{
		Name:        "email-service",
		MaxRequests: 3,                // сколько пробных запросов в HALF-OPEN
		Interval:    10 * time.Second, // сброс счетчиков
		Timeout:     30 * time.Second, // сколько ждать перед переходом в HALF-OPEN

		// Условие когда перейти в OPEN
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Открываем цепь если > 50% ошибок и было минимум 3 запроса
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.5
		},

		// Логируем изменения состояния
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Info().Msgf("Circuit Breaker '%s' changed from %v to %v", name, from, to)
		},
	}

	return &Handler{
		db:           db,
		cfg:          cfg,
		redis:        rds,
		metrics:      fiberprometheus.New("task-management-service"),
		emailBreaker: gobreaker.NewCircuitBreaker(settings),
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
