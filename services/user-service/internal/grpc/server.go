package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer *grpc.Server
	userHandler *UserHandler
	port       int
}

func NewGRPCServer(userHandler *UserHandler, port int) *Server {
	return &Server{
		grpcServer:  grpc.NewServer(),
		userHandler: userHandler,
		port:        port,
	}
}

func (s *Server) RegisterServices() {
	// Register user service handler
	// Note: In production, register actual protobuf-generated service
	// RegisterUserServiceServer(s.grpcServer, s.userHandler)

	// Enable reflection for development
	reflection.Register(s.grpcServer)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	fmt.Printf("gRPC server starting on port %d\n", s.port)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
