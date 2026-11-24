package service

import (
	"context"
	"io"
	"net"
	"strings"

	agentv1 "backend/api/agent/v1"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient 是后端到 Python Agent 的 gRPC 客户端。
type AgentClient struct {
	addr   string
	logger *log.Logger
}

func NewAgentClient(conf *viper.Viper, logger *log.Logger) *AgentClient {
	addr := conf.GetString("agent.addr")
	if host, _, err := net.SplitHostPort(addr); err == nil && !isLoopbackHost(host) {
		logger.Warn("Agent gRPC 未启用 TLS，凭据会以明文跨网络传输；请确保链路位于可信私网，或在网络层加密。",
			zap.String("agent_addr", addr))
	}
	return &AgentClient{addr: addr, logger: logger}
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (c *AgentClient) dial() (*grpc.ClientConn, error) {
	return grpc.NewClient(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func (c *AgentClient) Health(ctx context.Context) (*agentv1.HealthResponse, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return agentv1.NewAgentServiceClient(conn).Health(ctx, &agentv1.HealthRequest{})
}

// ListScripts 返回 Host 实际加载的脚本与运行时状态。
func (c *AgentClient) ListScripts(ctx context.Context) (*agentv1.ListScriptsResponse, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return agentv1.NewAgentServiceClient(conn).ListScripts(ctx, &agentv1.ListScriptsRequest{})
}

// Execute 下发任务并同步消费事件流；每条事件回调 onEvent。
func (c *AgentClient) Execute(ctx context.Context, req *agentv1.ExecuteRequest, onEvent func(*agentv1.ExecuteEvent)) error {
	conn, err := c.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	stream, err := agentv1.NewAgentServiceClient(conn).Execute(ctx, req)
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
func (c *AgentClient) Discover(ctx context.Context, req *agentv1.DiscoverRequest) (*agentv1.DiscoverResponse, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return agentv1.NewAgentServiceClient(conn).Discover(ctx, req)
}
