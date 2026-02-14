package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
)

// SetupRoutes func
func SetupRoutes(app *fiber.App) {
	app.Use(healthcheck.New())
}
