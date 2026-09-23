package logging

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Module = fx.Module("logging", fx.Provide(New))

type LevelController struct {
	level zap.AtomicLevel
}

type Result struct {
	fx.Out

	Logger          *zap.Logger
	LevelController *LevelController
}

func New(lifecycle fx.Lifecycle) (Result, error) {
	config := zap.NewProductionConfig()
	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	config.Level = level
	logger, err := config.Build()
	if err != nil {
		return Result{}, err
	}

	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		_ = logger.Sync()
		return nil
	}})
	return Result{Logger: logger, LevelController: &LevelController{level: level}}, nil
}

func NewLevelController(level zapcore.Level) *LevelController {
	return &LevelController{level: zap.NewAtomicLevelAt(level)}
}

func (controller *LevelController) Level() string { return controller.level.Level().String() }

func (controller *LevelController) SetLevel(value string) error {
	var level zapcore.Level
	if err := level.Set(strings.ToLower(strings.TrimSpace(value))); err != nil {
		return fmt.Errorf("invalid log level %q", value)
	}
	if level < zap.DebugLevel || level > zap.ErrorLevel {
		return fmt.Errorf("log level must be debug, info, warn, or error")
	}
	controller.level.SetLevel(level)
	return nil
}
