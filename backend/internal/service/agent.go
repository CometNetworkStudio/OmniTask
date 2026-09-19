package service

import (
	"context"
	"io"

	agentv1 "backend/api/agent/v1"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AgentClient 是后端到 Python Agent 的 gRPC 客户端。
type AgentClient struct {
	addr   string
	logger *log.Logger
}

func NewAgentClient(conf *viper.Viper, logger *log.Logger) *AgentClient {
	return &AgentClient{addr: conf.GetString("agent.addr"), logger: logger}
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
