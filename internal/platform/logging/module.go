package logging

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Module provides the application logger and directs Fx lifecycle events to
// the same JSON output. Keeping one logging pipeline makes container logs
// consistently queryable in Alloy and Loki.
var Module = fx.Module(
	"logging",
	fx.Provide(New),
	fx.WithLogger(NewFxEventLogger),
)

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

// NewFxEventLogger adapts the shared Zap logger to Fx's event logger. Fx emits
// stable fields such as constructor, module, runtime, and error into the JSON
// record instead of writing its default console-formatted messages.
func NewFxEventLogger(logger *zap.Logger) fxevent.Logger {
	fxLogger := &fxevent.ZapLogger{Logger: logger.Named("fx")}
	fxLogger.UseLogLevel(zap.InfoLevel)

	return fxLogger
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
