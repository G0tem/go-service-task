package factory

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/G0tem/go-service-task/internal/config"
	"github.com/G0tem/go-service-task/internal/handler"
	"github.com/G0tem/go-service-task/internal/router"
	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func httpService(cfg *config.Config) error {
	db, err := NewDB(cfg)
	if err != nil {
		return err
	}

	rds, err := NewRedis(cfg)
	if err != nil {
		return err
	}

	handlers := handler.NewHandler(db, rds, cfg)

	app := fiber.New(fiber.Config{})

	swaggerCfg := swagger.Config{
		BasePath: "/api/v1",
		FilePath: "./docs/swagger.yaml",
		Path:     "docs",
		CacheAge: 1,
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	app.Use(swagger.New(swaggerCfg))
	app.Use(fiberzerolog.New(fiberzerolog.Config{
		Logger: &logger,
	}))
	app.Use(cors.New())

	router.SetupRoutes(app)
	handlers.SetupRoutes(app)

	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	addr := fmt.Sprintf(":%v", cfg.HttpPort)
	log.Info().Str("addr", addr).Msg("starting HTTP server")
	err = app.Listen(addr)
	if err != nil {
		log.Error().Msgf("Unexpected error: %v", err)
		return err
	}
	log.Info().Msgf("Setup http port %v", cfg.HttpPort)

	return nil
}

func StartHttpService(cfg *config.Config) error {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(
		signalChannel,
		syscall.SIGUSR2, // Use for restart listening port
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGSEGV,
	)

	log.Info().Msgf("Setup http port %v", cfg.HttpPort)

	go httpService(cfg)

	for {
		signalEvent := <-signalChannel
		switch signalEvent {
		case syscall.SIGUSR2:
			time.Sleep(5 * time.Second)
			go httpService(cfg)
		case syscall.SIGQUIT,
			syscall.SIGTERM,
			syscall.SIGINT,
			syscall.SIGKILL:
			log.Error().Msgf("Signal event %q", signalEvent)
			return nil
		case syscall.SIGHUP:
			log.Error().Msgf("Signal event %q", signalEvent)
			return fmt.Errorf("signal hang up")
		case syscall.SIGSEGV:
			log.Error().Msgf("Signal event %q", signalEvent)
			return fmt.Errorf("segmentation violation")
		default:
			log.Error().Msgf("Unexpected signal %q", signalEvent)
		}
	}
}
