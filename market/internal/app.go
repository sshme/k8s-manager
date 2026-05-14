package internal

import (
	"context"
	"errors"
	"fmt"
	"k8s-manager/market/internal/presentation/grpc"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsPort is kept distinct from the gRPC port for dependency injection.
type MetricsPort int

// App represents the application
type App struct {
	grpcServer  *grpc.Server
	metricsPort int
}

// NewApp creates a new application instance
func NewApp(grpcServer *grpc.Server, metricsPort MetricsPort) *App {
	return &App{
		grpcServer:  grpcServer,
		metricsPort: int(metricsPort),
	}
}

// Run starts the application
func (a *App) Run() error {
	metricsServer := a.metricsServer()
	if metricsServer != nil {
		go func() {
			log.Printf("Prometheus metrics listening on port %d", a.metricsPort)
			if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("Metrics server error: %v", err)
			}
		}()
	}

	err := a.grpcServer.Run()

	if metricsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := metricsServer.Shutdown(ctx); shutdownErr != nil {
			log.Printf("Metrics server shutdown error: %v", shutdownErr)
		}
	}

	return err
}

func (a *App) metricsServer() *http.Server {
	if a.metricsPort <= 0 {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", a.metricsPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
