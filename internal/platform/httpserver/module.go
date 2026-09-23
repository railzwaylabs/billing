package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/platform/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Address string
	Name    string
}

type Server struct {
	server *http.Server
	logger *zap.Logger
}

var Module = fx.Module(
	"http_server",
	fx.Provide(NewEngine, NewServer),
	fx.Invoke(RegisterLifecycle),
)

func NewEngine(logger *zap.Logger, telemetry *metrics.Metrics) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), requestLogger(logger), telemetry.Middleware())
	return engine
}

func NewServer(config Config, engine *gin.Engine, logger *zap.Logger) *Server {
	return &Server{server: &http.Server{Addr: config.Address, Handler: engine, ReadHeaderTimeout: 10 * time.Second}, logger: logger.Named(config.Name)}
}

func RegisterLifecycle(lifecycle fx.Lifecycle, server *Server) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				server.logger.Info("HTTP server started", zap.String("address", server.server.Addr))
				if err := server.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					server.logger.Error("HTTP server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error { return server.server.Shutdown(ctx) },
	})
}

func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.Info("HTTP request", zap.String("method", c.Request.Method), zap.String("path", c.Request.URL.Path), zap.Int("status", c.Writer.Status()), zap.Duration("duration", time.Since(started)))
	}
}
