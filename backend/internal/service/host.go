package service

import (
	"context"
	"io"
	"net"
	"strings"

	hostv1 "backend/api/host/v1"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// HostClient 是后端到 Python 脚本 Host 的 gRPC 客户端。
type HostClient struct {
	addr   string
	logger *log.Logger
}

func NewHostClient(conf *viper.Viper, logger *log.Logger) *HostClient {
	addr := conf.GetString("host.addr")
	if host, _, err := net.SplitHostPort(addr); err == nil && !isLoopbackHost(host) {
		logger.Warn("Host gRPC 未启用 TLS，凭据会以明文跨网络传输；请确保链路位于可信私网，或在网络层加密。",
			zap.String("host_addr", addr))
	}
	return &HostClient{addr: addr, logger: logger}
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (c *HostClient) dial() (*grpc.ClientConn, error) {
	return grpc.NewClient(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (c *HostClient) Health(ctx context.Context) (*hostv1.HealthResponse, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return hostv1.NewHostServiceClient(conn).Health(ctx, &hostv1.HealthRequest{})
}

// ListScripts 返回 Host 实际加载的脚本与运行时状态。
func (c *HostClient) ListScripts(ctx context.Context) (*hostv1.ListScriptsResponse, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return hostv1.NewHostServiceClient(conn).ListScripts(ctx, &hostv1.ListScriptsRequest{})
}

// Execute 下发任务并同步消费事件流；每条事件回调 onEvent。
func (c *HostClient) Execute(ctx context.Context, req *hostv1.ExecuteRequest, onEvent func(*hostv1.ExecuteEvent)) error {
	conn, err := c.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	stream, err := hostv1.NewHostServiceClient(conn).Execute(ctx, req)
	if err != nil {
		return err
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		onEvent(event)
	}
}

// Discover 列出该账号在平台上的条目（脚本用 @listing 实现）。
func (c *HostClient) Discover(ctx context.Context, req *hostv1.DiscoverRequest) (*hostv1.DiscoverResponse, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return hostv1.NewHostServiceClient(conn).Discover(ctx, req)
}
