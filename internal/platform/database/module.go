package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Type     string
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	Sslmode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	LogLevel        string
	SlowThreshold   time.Duration
}

var Module = fx.Module("database", fx.Provide(NewDatabase))

// DSN returns the canonical PostgreSQL connection string shared by GORM and
// dedicated PostgreSQL connections such as the IAM LISTEN/NOTIFY listener.
func (config Config) DSN() string {
	host := config.Host
	if config.Port != "" {
		host = net.JoinHostPort(config.Host, config.Port)
	}
	query := url.Values{}
	query.Set("sslmode", config.Sslmode)
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.User, config.Password),
		Host:     host,
		Path:     "/" + config.Name,
		RawQuery: query.Encode(),
	}).String()
}

func NewDatabase(lc fx.Lifecycle, config Config, logger *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger: NewGORMLogger(logger.Named("gorm"), config.LogLevel, config.SlowThreshold),
	})
	if err != nil {
		return nil, fmt.Errorf("open billing database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get SQL database: %w", err)
	}

	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping billing database: %w", err)
	}

	logger.Info("configured database connection pool",
		zap.Int("max_open_connections", config.MaxOpenConns),
		zap.Int("max_idle_connections", config.MaxIdleConns),
		zap.Duration("connection_max_lifetime", config.ConnMaxLifetime),
		zap.Duration("connection_max_idle_time", config.ConnMaxIdleTime),
	)

	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		logger.Info("closing billing database")
		return sqlDB.Close()
	}})

	return db, nil
}
