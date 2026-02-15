package config

import (
	"os"
	"strconv"
	"time"

	"github.com/G0tem/go-service-task/internal"
)

type Config struct {
	LogLevel int    `default:"4" envconfig:"LOG_LEVEL"`
	HttpPort uint16 `default:"8001" envconfig:"HTTP_PORT"`

	SecretKey string `binding:"required" envconfig:"SECRET_KEY"`

	MysqlHost            string        `binding:"required" envconfig:"MYSQL_HOST"`
	MysqlPort            string        `binding:"required" envconfig:"MYSQL_PORT"`
	MysqlDb              string        `binding:"required" envconfig:"MYSQL_DATABASE"`
	MysqlUser            string        `binding:"required" envconfig:"MYSQL_USER"`
	MysqlPassword        string        `binding:"required" envconfig:"MYSQL_PASSWORD"`
	MysqlMaxIdleConns    int           `default:"10" envconfig:"MYSQL_MAX_IDLE_CONNS"`
	MysqlMaxOpenConns    int           `default:"100" envconfig:"MYSQL_MAX_OPEN_CONNS"`
	MysqlConnMaxLifetime time.Duration `default:"1h" envconfig:"MYSQL_CONN_MAX_LIFETIME"`

	RedisAddr string `binding:"required" envconfig:"REDIS_ADDR"`
	RedisDB   int    `binding:"required" envconfig:"REDIS_DB"`
}

func LoadConfig() Config {
	logLevel, _ := strconv.Atoi(os.Getenv("LOG_LEVEL"))

	return Config{
		LogLevel: logLevel,
		HttpPort: internal.ParseUint16(os.Getenv("HTTP_PORT"), 8001),

		SecretKey: os.Getenv("SECRET_KEY"),

		MysqlHost:            os.Getenv("MYSQL_HOST"),
		MysqlPort:            os.Getenv("MYSQL_PORT"),
		MysqlDb:              os.Getenv("MYSQL_DATABASE"),
		MysqlUser:            os.Getenv("MYSQL_USER"),
		MysqlPassword:        os.Getenv("MYSQL_PASSWORD"),
		MysqlMaxIdleConns:    internal.ParseInt(os.Getenv("MYSQL_MAX_IDLE_CONNS"), 10),
		MysqlMaxOpenConns:    internal.ParseInt(os.Getenv("MYSQL_MAX_OPEN_CONNS"), 100),
		MysqlConnMaxLifetime: internal.ParseDuration(os.Getenv("MYSQL_CONN_MAX_LIFETIME"), 1*time.Hour),

		RedisAddr: os.Getenv("REDIS_ADDR"),
		RedisDB:   internal.ParseInt(os.Getenv("REDIS_DB"), 0),
	}
}
