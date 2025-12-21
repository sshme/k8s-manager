package internal

import (
	"k8s-manager/market/internal/presentation/grpc"
)

// App represents the application
type App struct {
	grpcServer *grpc.Server
}

// NewApp creates a new application instance
func NewApp(grpcServer *grpc.Server) *App {
	return &App{
		grpcServer: grpcServer,
	}
}

// Run starts the application
func (a *App) Run() error {
	return a.grpcServer.Run()
}

