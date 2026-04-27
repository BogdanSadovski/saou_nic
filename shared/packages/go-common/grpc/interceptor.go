package grpcc

import (
	"context"
	"time"

	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingUnaryInterceptor returns a unary client interceptor that logs requests and responses.
func LoggingUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		logger.Debug("gRPC request",
			zap.String("method", method),
			zap.Any("request", req),
		)

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)

		if err != nil {
			logger.Warn("gRPC request failed",
				zap.String("method", method),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			logger.Debug("gRPC response",
				zap.String("method", method),
				zap.Duration("duration", duration),
				zap.Any("response", reply),
			)
		}

		return err
	}
}

// LoggingStreamInterceptor returns a stream client interceptor that logs stream operations.
func LoggingStreamInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()

		logger.Debug("gRPC stream started",
			zap.String("method", method),
			zap.Bool("client_streaming", desc.ClientStreams),
			zap.Bool("server_streaming", desc.ServerStreams),
		)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			logger.Warn("gRPC stream failed to start",
				zap.String("method", method),
				zap.Duration("setup_duration", time.Since(start)),
				zap.Error(err),
			)
			return nil, err
		}

		return &loggedClientStream{
			ClientStream: stream,
			method:       method,
			start:        start,
		}, nil
	}
}

// loggedClientStream wraps a ClientStream with logging.
type loggedClientStream struct {
	grpc.ClientStream
	method string
	start  time.Time
}

func (s *loggedClientStream) SendMsg(msg interface{}) error {
	err := s.ClientStream.SendMsg(msg)
	if err != nil {
		logger.Warn("gRPC stream send failed",
			zap.String("method", s.method),
			zap.Duration("elapsed", time.Since(s.start)),
			zap.Error(err),
		)
	}
	return err
}

func (s *loggedClientStream) RecvMsg(msg interface{}) error {
	err := s.ClientStream.RecvMsg(msg)
	if err != nil {
		logger.Warn("gRPC stream recv failed",
			zap.String("method", s.method),
			zap.Duration("elapsed", time.Since(s.start)),
			zap.Error(err),
		)
	}
	return err
}

// ErrorHandlingUnaryInterceptor returns a unary client interceptor that handles gRPC errors.
func ErrorHandlingUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			st := status.Convert(err)
			logger.Error("gRPC error",
				zap.String("method", method),
				zap.String("code", st.Code().String()),
				zap.String("message", st.Message()),
			)
		}
		return err
	}
}

// RequestIDUnaryInterceptor returns a unary client interceptor that injects request ID.
func RequestIDUnaryInterceptor(requestID string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md := metadata.Pairs("x-request-id", requestID)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// TimeoutUnaryInterceptor returns a unary client interceptor that adds a timeout.
func TimeoutUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// RetryUnaryInterceptor returns a unary client interceptor that retries on certain errors.
func RetryUnaryInterceptor(maxRetries int, retryableCodes ...codes.Code) grpc.UnaryClientInterceptor {
	codeMap := make(map[codes.Code]bool)
	for _, code := range retryableCodes {
		codeMap[code] = true
	}
	if len(codeMap) == 0 {
		codeMap[codes.Unavailable] = true
		codeMap[codes.DeadlineExceeded] = true
	}

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var lastErr error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(attempt) * 100 * time.Millisecond
				logger.Debug("retrying gRPC request",
					zap.String("method", method),
					zap.Int("attempt", attempt),
					zap.Duration("backoff", backoff),
				)
				time.Sleep(backoff)
			}

			lastErr = invoker(ctx, method, req, reply, cc, opts...)
			if lastErr == nil {
				return nil
			}

			st := status.Convert(lastErr)
			if !codeMap[st.Code()] {
				return lastErr
			}
		}
		return lastErr
	}
}

// ChainUnaryInterceptors chains multiple unary client interceptors.
func ChainUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) grpc.UnaryClientInterceptor {
	n := len(interceptors)
	if n == 0 {
		return func(
			ctx context.Context,
			method string,
			req, reply interface{},
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var handler grpc.UnaryInvoker = func(
			currentCtx context.Context,
			currentMethod string,
			currentReq, currentReply interface{},
			currentConn *grpc.ClientConn,
			currentOpts ...grpc.CallOption,
		) error {
			return invoker(currentCtx, currentMethod, currentReq, currentReply, currentConn, currentOpts...)
		}

		for i := n - 1; i >= 0; i-- {
			i := i
			next := handler
			interceptor := interceptors[i]
			handler = func(
				currentCtx context.Context,
				currentMethod string,
				currentReq, currentReply interface{},
				currentConn *grpc.ClientConn,
				currentOpts ...grpc.CallOption,
			) error {
				return interceptor(currentCtx, currentMethod, currentReq, currentReply, currentConn, next, currentOpts...)
			}
		}

		return handler(ctx, method, req, reply, cc, opts...)
	}
}
