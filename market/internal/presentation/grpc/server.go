package grpc

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	authOIDC "k8s-manager/market/internal/infrastructure/auth"
	"k8s-manager/market/internal/presentation/grpc/auth"
	"k8s-manager/market/internal/presentation/grpc/plugin"
	"k8s-manager/market/internal/presentation/grpc/user"
	marketv1 "k8s-manager/proto/gen/v1/market"
	usersv1 "k8s-manager/proto/gen/v1/users"
)

// Server wraps the gRPC server
type Server struct {
	grpcServer *grpc.Server
	port       int
}

// NewServer creates a new gRPC server
func NewServer(port int, userHandler *user.Handler, pluginHandler *plugin.Handler) *Server {
	oidcClient := authOIDC.NewOIDCClient(
		"http://localhost:8080",
		"my-client-id",
		true, // только для dev
	)

	rules := map[string]auth.Rule{
		"/market.v1.PluginService/ListPlugins": {
			Public: true,
		},
		"/market.v1.PluginService/CreatePlugin": {
			Roles: []string{"admin", "editor"},
		},
		"/market.v1.PluginService/DeletePlugin": {
			Roles: []string{"admin"},
		},
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(rules, auth.NewOIDCTokenParser(oidcClient))),
	)

	usersv1.RegisterUserServiceServer(grpcServer, userHandler)
	marketv1.RegisterPluginServiceServer(grpcServer, pluginHandler)

	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	log.Printf("gRPC server listening on port %d", s.port)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// Run starts the server and handles graceful shutdown
func (s *Server) Run() error {
	errChan := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		log.Printf("Received signal: %v, shutting down gracefully...", sig)
		s.Stop()
		return nil
	}
}
