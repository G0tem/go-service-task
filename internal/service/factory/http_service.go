package factory

import (
	"context"
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

func httpService(cfg *config.Config) (*fiber.App, error) {
	db, err := NewDB(cfg)
	if err != nil {
		return nil, err
	}

	rds, err := NewRedis(cfg)
	if err != nil {
		return nil, err
	}

	handlers := handler.NewHandler(db, rds, cfg)

	app := fiber.New(fiber.Config{
		// Добавляем настройки для graceful shutdown
		DisableStartupMessage: false,
	})

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

	return app, nil
}

func StartHttpService(cfg *config.Config) error {
	serverErrors := make(chan error, 1)
	signalChannel := make(chan os.Signal, 1)

	signal.Notify(
		signalChannel,
		syscall.SIGUSR2,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGSEGV,
	)

	log.Info().Msgf("Setup http port %v", cfg.HttpPort)

	// Создаем сервер
	app, err := httpService(cfg)
	if err != nil {
		return err
	}

	// Запускаем сервер
	addr := fmt.Sprintf(":%v", cfg.HttpPort)
	go func() {
		log.Info().Str("addr", addr).Msg("starting HTTP server")
		if err := app.Listen(addr); err != nil {
			serverErrors <- err
		}
	}()

	// Ожидаем сигналы или ошибки сервера
	for {
		select {
		case err := <-serverErrors:
			// Критическая ошибка сервера
			log.Error().Err(err).Msg("HTTP server error")
			return err

		case signalEvent := <-signalChannel:
			switch signalEvent {
			case syscall.SIGUSR2:
				log.Info().Msg("Received SIGUSR2, restarting server...")

				// Graceful shutdown текущего сервера
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := app.ShutdownWithContext(ctx); err != nil {
					log.Error().Err(err).Msg("Error during server shutdown")
				}

				// Запускаем новый сервер
				time.Sleep(5 * time.Second)

				newApp, err := httpService(cfg)
				if err != nil {
					log.Error().Err(err).Msg("Failed to create new server instance")
					continue
				}

				// Обновляем ссылку
				app = newApp

				// Запускаем новый сервер
				go func() {
					if err := app.Listen(addr); err != nil {
						serverErrors <- err
					}
				}()

			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				log.Info().Msgf("Received signal %q, starting graceful shutdown...", signalEvent)

				// Создаем контекст с таймаутом для graceful shutdown
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// Пытаемся gracefully завершить сервер
				if err := app.ShutdownWithContext(ctx); err != nil {
					log.Error().Err(err).Msg("Error during server shutdown")
					return err
				}

				log.Info().Msg("HTTP server gracefully stopped")
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
}
