package database

import (
	"fmt"
	"strings"

	"github.com/G0tem/go-service-task/internal"
	"github.com/G0tem/go-service-task/internal/config"
	"github.com/G0tem/go-service-task/internal/model"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Connect function db
func Connect(cfg *config.Config) (*gorm.DB, error) {
	mysqlPort := cfg.MysqlPort
	port := internal.ParseInt(mysqlPort, 3306)

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MysqlUser,
		cfg.MysqlPassword,
		cfg.MysqlHost,
		port,
		cfg.MysqlDb,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.LogLevel(cfg.LogLevel)),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",                                // убираем префикс
			SingularTable: false,                             // use plural table name, table for `User` would be `users`
			NoLowerCase:   false,                             // skip the snake_casing of names
			NameReplacer:  strings.NewReplacer("CID", "Cid"), // use name replacer to change struct/field name before convert it to db name
		},
	})

	if err != nil {
		log.Error().Msgf("failed to connect to database. %v\n", err)
		return nil, err
	}

	log.Info().Msg("running migrations")
	err = db.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.Task{},
		&model.TaskHistory{},
		&model.TaskComment{},
	)
	if err != nil {
		log.Error().Msgf("failed run auto-migrations. %v\n", err)
		return nil, err
	}

	// Apply connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Error().Msgf("failed to create database connection pool. %v\n", err)
		return nil, err
	}
	sqlDB.SetMaxIdleConns(cfg.MysqlMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MysqlMaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.MysqlConnMaxLifetime)

	return db, nil
}
