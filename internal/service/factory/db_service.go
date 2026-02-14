package factory

import (
	"context"
	"fmt"
	"sync"

	"github.com/G0tem/go-service-task/internal/config"
	"github.com/G0tem/go-service-task/internal/database"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

var (
	// Для GORM
	databaseServiceInstance *gorm.DB
	databaseOnce            sync.Once
	databaseErr             error

	// Для Redis
	redisClientInstance *redis.Client
	redisOnce           sync.Once
	redisErr            error
)

func NewDB(cfg *config.Config) (*gorm.DB, error) {
	databaseOnce.Do(func() {
		databaseServiceInstance, databaseErr = database.Connect(cfg)
	})
	return databaseServiceInstance, databaseErr
}

func NewRedis(cfg *config.Config) (*redis.Client, error) {
	redisOnce.Do(func() {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: "",
			DB:       cfg.RedisDB,
		})

		ctx := context.Background()
		if err := client.Ping(ctx).Err(); err != nil {
			redisErr = fmt.Errorf("failed to connect to Redis: %w", err)
			log.Warn().Msgf("[NewRedis] failed to connect to Redis, addr: %s", cfg.RedisAddr)
			return
		}

		redisClientInstance = client
		log.Info().Msgf("[NewRedis] create redisConnect")
	})

	if redisErr != nil {
		return nil, redisErr
	}
	return redisClientInstance, nil
}
