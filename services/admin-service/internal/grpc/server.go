package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/real-ass/admin-service/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server and its dependencies.
type Server struct {
	server        *grpc.Server
	adminHandler  *AdminHandler
	config        *config.GRPCConfig
}

// NewServer creates a new gRPC server.
func NewServer(
	adminHandler *AdminHandler,
	cfg *config.GRPCConfig,
) *Server {
	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)

	s := &Server{
		server:       grpcServer,
		adminHandler: adminHandler,
		config:       cfg,
	}

	// Register services
	RegisterAdminServiceServer(grpcServer, adminHandler)

	// Enable reflection for development/debugging
	reflection.Register(grpcServer)

	return s
}

// Start starts the gRPC server.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	fmt.Printf("gRPC server starting on %s\n", addr)
	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}

	return nil
}

// GracefulStop gracefully stops the gRPC server.
func (s *Server) GracefulStop() {
	s.server.GracefulStop()
}

// unaryInterceptor is a unary interceptor for logging and error handling.
func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Log the request
	fmt.Printf("[gRPC] Unary request: %s\n", info.FullMethod)

	// Call the handler
	resp, err := handler(ctx, req)
	if err != nil {
		fmt.Printf("[gRPC] Unary error: %s - %v\n", info.FullMethod, err)
	}

	return resp, err
}

// streamInterceptor is a stream interceptor for logging and error handling.
func streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	fmt.Printf("[gRPC] Stream request: %s\n", info.FullMethod)
	return handler(srv, ss)
}

// ==================== Proto Definitions (inline for simplicity) ====================
// In production, these would be generated from .proto files using protoc.

// AdminServiceServer is the gRPC service interface for admin operations.
type AdminServiceServer interface {
	// MustEmbedUnimplementedAdminServiceServer()
}

// RegisterAdminServiceServer registers the admin service with the gRPC server.
func RegisterAdminServiceServer(s *grpc.Server, srv AdminServiceServer) {
	// In production, use generated code:
	// pb.RegisterAdminServiceServer(s, srv)
}

// RequestIDInterceptor adds request ID to context.
func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Add request ID to context (in production, extract from metadata)
		return handler(ctx, req)
	}
}
