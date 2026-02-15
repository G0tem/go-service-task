package main

import (
	"os"
	"runtime"

	_ "github.com/G0tem/go-service-task/docs" // swagger docs
	"github.com/G0tem/go-service-task/internal/config"
	"github.com/G0tem/go-service-task/internal/service/factory"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// @title Service task  Swagger
// @version 1.0
// @description This is an API of auth-service
// @schemes http https

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Description for what is this security definition being used

// @BasePath /api/v1
func main() {
	// Initialize Zerolog logger with output to stdout
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.LoadConfig()
	zerolog.SetGlobalLevel(zerolog.Level(cfg.LogLevel))

	limitToTwoThreads()

	err := factory.StartHttpService(&cfg)

	if err != nil {
		log.Error().Msgf("Attempt to start application fail with error %v", err)
	}
}

func limitToTwoThreads() {
	currentThreadsCount := runtime.GOMAXPROCS(0)

	// Если больше 2, уменьшаем до 2
	if currentThreadsCount > 2 {
		runtime.GOMAXPROCS(2)
	}
}
