package metrics

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Address        string
	Service        string
	OrganizationID string
	ProjectID      string
}

type Metrics struct {
	Handler  http.Handler
	Requests *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

type Server struct {
	server   *http.Server
	logger   *zap.Logger
	listener net.Listener
}

var Module = fx.Module("metrics", fx.Provide(New, NewServer), fx.Invoke(RegisterLifecycle))

func New(config Config) *Metrics {
	labels := prometheus.Labels{
		"service":         config.Service,
		"organization_id": config.OrganizationID,
		"project_id":      config.ProjectID,
	}
	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWith(labels, registry)
	registerer.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "billing",
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "billing",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
	registerer.MustRegister(requests, duration)

	return &Metrics{
		Handler:  promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		Requests: requests,
		Duration: duration,
	}
}

func NewServer(config Config, metrics *Metrics, logger *zap.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.Handler)
	mux.HandleFunc("GET /healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /readyz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	return &Server{
		server: &http.Server{Addr: config.Address, Handler: mux, ReadHeaderTimeout: 5 * time.Second},
		logger: logger.Named(config.Service + "_metrics"),
	}
}

func RegisterLifecycle(lifecycle fx.Lifecycle, server *Server) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", server.server.Addr)
			if err != nil {
				return err
			}
			server.listener = listener
			go func() {
				server.logger.Info("metrics server started", zap.String("address", server.server.Addr))
				if err := server.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					server.logger.Error("metrics server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error { return server.server.Shutdown(ctx) },
	})
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		m.Requests.WithLabelValues(c.Request.Method, route, status).Inc()
		m.Duration.WithLabelValues(c.Request.Method, route, status).Observe(time.Since(started).Seconds())
	}
}
