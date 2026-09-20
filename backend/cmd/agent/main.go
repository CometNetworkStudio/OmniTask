package main

import (
	"context"
	"flag"
	"fmt"

	agentv1 "backend/api/agent/v1"
	"backend/internal/agenthost"
	"backend/pkg/config"
	"backend/pkg/log"
	grpcserver "backend/pkg/server/grpc"
	"go.uber.org/zap"
)

// OmniTask Agent Host：Go 脚本宿主，以子进程 + stdio JSON 运行平台脚本。
func main() {
	var envConf = flag.String("conf", "config/deploy.yml", "config path, eg: -conf ./config/deploy.yml")
	flag.Parse()
	conf := config.NewConfig(*envConf)
	logger := log.NewLog(conf)

	host, err := agenthost.NewHost(conf, logger)
	if err != nil {
		panic(err)
	}

	srv := grpcserver.NewServer(
		logger,
		grpcserver.WithServerHost(conf.GetString("agent.host")),
		grpcserver.WithServerPort(conf.GetInt("agent.port")),
	)
	agentv1.RegisterAgentServiceServer(srv.Server, host)

	logger.Info("agent host start",
		zap.String("addr", fmt.Sprintf("%s:%d", conf.GetString("agent.host"), conf.GetInt("agent.port"))),
		zap.String("scripts_dir", conf.GetString("agent.scripts_dir")),
	)
	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}
}
