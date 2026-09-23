package pprof

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"time"

	"github.com/railzwaylabs/billing/internal/platform/logging"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Address string
	Service string
}

type Server struct {
	server   *http.Server
	mux      *http.ServeMux
	logger   *zap.Logger
	listener net.Listener
}

var Module = fx.Module("pprof", fx.Provide(NewServer), fx.Invoke(RegisterLifecycle))

// AdminModule adds runtime log-level management to the management listener.
// It must only be wired into cmd/admin-api.
var AdminModule = fx.Module("pprof_admin", fx.Invoke(RegisterLogMode))

func NewServer(config Config, logger *zap.Logger) *Server {
	mux := http.NewServeMux()
	registerPprof(mux)
	return &Server{
		server: &http.Server{
			Addr:              config.Address,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		mux:    mux,
		logger: logger.Named(config.Service + "_management"),
	}
}

func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", httppprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", httppprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", httppprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", httppprof.Symbol)
	mux.HandleFunc("POST /debug/pprof/symbol", httppprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", httppprof.Trace)
	for _, profile := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		mux.Handle("GET /debug/pprof/"+profile, httppprof.Handler(profile))
	}
}

func RegisterLogMode(server *Server, controller *logging.LevelController) {
	server.mux.HandleFunc("GET /log/mode", func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{"level": controller.Level()})
	})
	server.mux.HandleFunc("PUT /log/mode", func(writer http.ResponseWriter, request *http.Request) {
		var body struct {
			Level string `json:"level"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, 1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&body); err != nil {
			writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if err := controller.SetLevel(body.Level); err != nil {
			writeJSON(writer, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"level": controller.Level()})
	})
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
				server.logger.Info("management server started", zap.String("address", server.server.Addr))
				if err := server.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					server.logger.Error("management server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error { return server.server.Shutdown(ctx) },
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
