package grpc

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

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
		getEnv("KEYCLOAK_ISSUER_URL", "http://localhost:8081/realms/k8s-manager"),
		getEnv("KEYCLOAK_CLIENT_ID", "k8s-manager-cli"),
		getEnvBool("KEYCLOAK_INSECURE_SKIP_TLS", false),
		getEnv("KEYCLOAK_TOKEN_ISSUER_URL", ""),
	)

	rules := map[string]auth.Rule{
		"/grpc.health.v1.Health/Check": {Public: true},
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo":      {Public: true},
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo": {Public: true},
		"/market.v1.PluginService/CreatePlugin": {
			RequiredRoles: []string{auth.RoleMarketAdmin, auth.RoleMarketPublisher},
		},
		"/market.v1.PluginService/UpdatePlugin": {
			RequiredRoles: []string{auth.RoleMarketAdmin, auth.RoleMarketPublisher},
		},
		"/market.v1.PluginService/UpdatePluginStatus": {
			RequiredRoles: []string{auth.RoleMarketAdmin},
		},
		"/market.v1.PluginService/UpdatePluginTrustStatus": {
			RequiredRoles: []string{auth.RoleMarketAdmin},
		},
		"/market.v1.PluginService/CreateRelease": {
			RequiredRoles: []string{auth.RoleMarketAdmin, auth.RoleMarketPublisher},
		},
		"/market.v1.PluginService/UploadArtifact": {
			RequiredRoles: []string{auth.RoleMarketAdmin, auth.RoleMarketPublisher},
		},
		"/market.v1.PluginService/DeleteArtifact": {
			RequiredRoles: []string{auth.RoleMarketAdmin, auth.RoleMarketPublisher},
		},
		"/market.v1.PublisherService/CreatePublisher": {
			RequiredRoles: []string{auth.RoleMarketAdmin, auth.RoleMarketPublisher},
		},
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(auth.UnaryAuthInterceptor(rules, auth.NewOIDCTokenParser(oidcClient))),
		grpc.StreamInterceptor(auth.StreamAuthInterceptor(rules, auth.NewOIDCTokenParser(oidcClient))),
	)
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("market.v1.PluginService", healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("market.v1.PublisherService", healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("users.v1.UserService", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)

	usersv1.RegisterUserServiceServer(grpcServer, userHandler)
	marketv1.RegisterPluginServiceServer(grpcServer, pluginHandler)
	marketv1.RegisterPublisherServiceServer(grpcServer, pluginHandler)
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		port:       port,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
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
