package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/railzwaylabs/billing/internal/platform/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Address            string
	Name               string
	CORSAllowedOrigins string
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

func NewEngine(config Config, logger *zap.Logger, telemetry *metrics.Metrics) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), corsMiddleware(config.CORSAllowedOrigins), requestLogger(logger), telemetry.Middleware())
	return engine
}

func corsMiddleware(configured string) gin.HandlerFunc {
	allowed := make(map[string]struct{})
	for _, origin := range strings.Split(configured, ",") {
		if value := strings.TrimSpace(origin); value != "" {
			allowed[value] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		_, accepted := allowed[origin]
		if origin != "" && accepted {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			if origin == "" || !accepted {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
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
