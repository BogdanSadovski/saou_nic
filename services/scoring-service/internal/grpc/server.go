package grpcserver

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server and its dependencies.
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

// NewServer creates a new gRPC server instance.
func NewServer(addr string, scoringHandler *ScoringHandler) (*Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("create listener: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			loggingInterceptor,
			recoveryInterceptor,
		),
	)

	// Register the scoring service
	RegisterScoringServiceServer(grpcServer, scoringHandler)

	// Enable reflection for development/debugging
	reflection.Register(grpcServer)

	return &Server{
		grpcServer: grpcServer,
		listener:   lis,
	}, nil
}

// Start begins serving gRPC requests.
func (s *Server) Start() error {
	return s.grpcServer.Serve(s.listener)
}

// GracefulStop stops the gRPC server gracefully.
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}

// loggingInterceptor logs incoming gRPC requests.
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// TODO: Add proper logging integration
	// log.Printf("gRPC request: %s", info.FullMethod)
	return handler(ctx, req)
}

// recoveryInterceptor recovers from panics in gRPC handlers.
func recoveryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal server error: %v", r)
		}
	}()
	return handler(ctx, req)
}
