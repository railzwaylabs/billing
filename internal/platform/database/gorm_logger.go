package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const defaultSlowQueryThreshold = 200 * time.Millisecond

// GORMLogger writes database events through Zap as structured JSON. It also
// implements GORM's parameter filter so query arguments (credentials, tokens,
// customer data, and other values) are not interpolated into application logs.
type GORMLogger struct {
	logger        *zap.Logger
	level         gormlogger.LogLevel
	slowThreshold time.Duration
}

var (
	_ gormlogger.Interface = (*GORMLogger)(nil)
	_ gorm.ParamsFilter    = (*GORMLogger)(nil)
)

// NewGORMLogger constructs the database logger using silent, error, warn, or
// info verbosity. Unknown values deliberately fall back to warn.
func NewGORMLogger(logger *zap.Logger, level string, slowThreshold time.Duration) *GORMLogger {
	if slowThreshold <= 0 {
		slowThreshold = defaultSlowQueryThreshold
	}

	return &GORMLogger{
		logger:        logger,
		level:         parseGORMLogLevel(level),
		slowThreshold: slowThreshold,
	}
}

// LogMode returns an independent logger value with the requested GORM level.
func (logger *GORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *logger
	clone.level = level

	return &clone
}

// Info records a GORM informational event as JSON.
func (logger *GORMLogger) Info(_ context.Context, message string, args ...interface{}) {
	if logger.level < gormlogger.Info {
		return
	}
	logger.logger.Info("gorm info", zap.String("detail", fmt.Sprintf(message, args...)))
}

// Warn records a GORM warning event as JSON.
func (logger *GORMLogger) Warn(_ context.Context, message string, args ...interface{}) {
	if logger.level < gormlogger.Warn {
		return
	}
	logger.logger.Warn("gorm warning", zap.String("detail", fmt.Sprintf(message, args...)))
}

// Error records a GORM error event as JSON.
func (logger *GORMLogger) Error(_ context.Context, message string, args ...interface{}) {
	if logger.level < gormlogger.Error {
		return
	}
	logger.logger.Error("gorm error", zap.String("detail", fmt.Sprintf(message, args...)))
}

// Trace records failed and slow queries, or every query when log level is
// info. SQL parameters remain placeholders because ParamsFilter strips values
// before GORM invokes the trace callback.
func (logger *GORMLogger) Trace(_ context.Context, startedAt time.Time, query func() (string, int64), queryErr error) {
	if logger.level == gormlogger.Silent {
		return
	}

	elapsed := time.Since(startedAt)
	sql, rows := query()
	fields := []zap.Field{
		zap.String("sql", strings.TrimSpace(sql)),
		zap.Int64("rows_affected", rows),
		zap.Int64("duration_ms", elapsed.Milliseconds()),
	}

	if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) && logger.level >= gormlogger.Error {
		logger.logger.Error("database query failed", append(fields, zap.Error(queryErr))...)
		return
	}

	if elapsed > logger.slowThreshold && logger.level >= gormlogger.Warn {
		logger.logger.Warn("slow database query", append(fields,
			zap.Int64("slow_threshold_ms", logger.slowThreshold.Milliseconds()),
		)...)
		return
	}

	if logger.level >= gormlogger.Info {
		logger.logger.Info("database query completed", fields...)
	}
}

// ParamsFilter retains the SQL statement but removes its bind values before
// GORM renders the statement passed to Trace.
func (logger *GORMLogger) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func parseGORMLogLevel(value string) gormlogger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Warn
	}
}
