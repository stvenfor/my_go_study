package grpcdelivery

import (
	"fmt"
	"net"

	analyticsv1 "github.com/stvenfor/my_go_study/api/gen/go/analytics/v1"
	grpcauth "github.com/stvenfor/my_go_study/internal/delivery/grpc/interceptor"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server 包装 grpc.Server 与 listener。
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	log        *zap.Logger
	port       int
}

// Options gRPC 启动依赖。
type Options struct {
	Port          int
	Log           *zap.Logger
	Authenticator *grpcauth.Authenticator
	AnalyticsUC   *usecase.AnalyticsUsecase
}

// NewServer 创建并注册服务（尚未 Listen）。
func NewServer(opts Options) (*Server, error) {
	if opts.Port <= 0 {
		return nil, fmt.Errorf("无效的 gRPC 端口: %d", opts.Port)
	}
	if opts.Authenticator == nil {
		return nil, fmt.Errorf("缺少 gRPC Authenticator")
	}
	if opts.AnalyticsUC == nil {
		return nil, fmt.Errorf("缺少 AnalyticsUsecase")
	}
	log := opts.Log
	if log == nil {
		log = zap.NewNop()
	}

	gs := grpc.NewServer(
		grpc.UnaryInterceptor(opts.Authenticator.UnaryServerInterceptor()),
	)
	analyticsv1.RegisterAnalyticsServiceServer(gs, NewAnalyticsServer(opts.AnalyticsUC))
	reflection.Register(gs)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", opts.Port))
	if err != nil {
		return nil, fmt.Errorf("监听 gRPC 端口失败: %w", err)
	}
	return &Server{grpcServer: gs, listener: lis, log: log, port: opts.Port}, nil
}

// Serve 阻塞服务（应在 goroutine 中调用）。
func (s *Server) Serve() error {
	s.log.Info("gRPC 服务启动", zap.Int("port", s.port))
	return s.grpcServer.Serve(s.listener)
}

// GracefulStop 优雅停止。
func (s *Server) GracefulStop() {
	if s == nil || s.grpcServer == nil {
		return
	}
	s.grpcServer.GracefulStop()
}
