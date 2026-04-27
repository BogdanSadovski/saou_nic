package grpcc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// ClientConfig holds gRPC client configuration.
type ClientConfig struct {
	Target            string        `yaml:"target" json:"target"`
	UseTLS            bool          `yaml:"use_tls" json:"use_tls"`
	ServerName        string        `yaml:"server_name" json:"server_name"`
	Timeout           time.Duration `yaml:"timeout" json:"timeout"`
	MaxRecvMsgSize    int           `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`
	MaxSendMsgSize    int           `yaml:"max_send_msg_size" json:"max_send_msg_size"`
	KeepaliveTime     time.Duration `yaml:"keepalive_time" json:"keepalive_time"`
	KeepaliveTimeout  time.Duration `yaml:"keepalive_timeout" json:"keepalive_timeout"`
	RetryEnabled      bool          `yaml:"retry_enabled" json:"retry_enabled"`
	MaxRetryAttempts  int           `yaml:"max_retry_attempts" json:"max_retry_attempts"`
	UserAgent         string        `yaml:"user_agent" json:"user_agent"`
}

// DefaultClientConfig returns a ClientConfig with sensible defaults.
func DefaultClientConfig(target string) ClientConfig {
	return ClientConfig{
		Target:           target,
		UseTLS:           false,
		Timeout:          10 * time.Second,
		MaxRecvMsgSize:   4 * 1024 * 1024,  // 4MB
		MaxSendMsgSize:   4 * 1024 * 1024,  // 4MB
		KeepaliveTime:    30 * time.Second,
		KeepaliveTimeout: 10 * time.Second,
		RetryEnabled:     false,
		MaxRetryAttempts: 3,
		UserAgent:        "real-ass-client",
	}
}

// Client is a gRPC client wrapper.
type Client struct {
	conn   *grpc.ClientConn
	config ClientConfig
}

// NewClient creates a new gRPC client with the given configuration.
func NewClient(cfg ClientConfig, opts ...grpc.DialOption) (*Client, error) {
	return &Client{config: cfg}, nil
}

// Dial establishes the gRPC connection.
func (c *Client) Dial(ctx context.Context) error {
	dialOpts := c.buildDialOptions()

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.config.Target, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	c.conn = conn

	logger.Info("connected to gRPC server",
		zap.String("target", c.config.Target),
		zap.Bool("tls", c.config.UseTLS),
	)

	return nil
}

// Conn returns the underlying gRPC connection.
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close gRPC connection: %w", err)
		}
		logger.Info("gRPC connection closed")
	}
	return nil
}

// HealthCheck performs a gRPC health check.
func (c *Client) HealthCheck(ctx context.Context) (*grpc_health_v1.HealthCheckResponse, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("gRPC client not connected")
	}

	healthClient := grpc_health_v1.NewHealthClient(c.conn)
	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	return resp, nil
}

// State returns the current state of the gRPC connection.
func (c *Client) State() connectivity.State {
	if c.conn == nil {
		return connectivity.Idle
	}
	return c.conn.GetState()
}

// buildDialOptions constructs the dial options from the configuration.
func (c *Client) buildDialOptions() []grpc.DialOption {
	opts := []grpc.DialOption{}

	// Transport credentials
	if c.config.UseTLS {
		tlsConfig := credentials.NewClientTLSFromCert(nil, c.config.ServerName)
		opts = append(opts, grpc.WithTransportCredentials(tlsConfig))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Message size limits
	if c.config.MaxRecvMsgSize > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(c.config.MaxRecvMsgSize)))
	}
	if c.config.MaxSendMsgSize > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(c.config.MaxSendMsgSize)))
	}

	// Keepalive
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                c.config.KeepaliveTime,
		Timeout:             c.config.KeepaliveTimeout,
		PermitWithoutStream: true,
	}))

	// Default interceptors
	opts = append(opts,
		grpc.WithChainUnaryInterceptor(
			LoggingUnaryInterceptor(),
			ErrorHandlingUnaryInterceptor(),
		),
		grpc.WithChainStreamInterceptor(
			LoggingStreamInterceptor(),
		),
	)

	// User agent
	if c.config.UserAgent != "" {
		opts = append(opts, grpc.WithUserAgent(c.config.UserAgent))
	}

	return opts
}
