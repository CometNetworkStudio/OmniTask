package main

import (
	"context"
	"flag"
	"fmt"

	hostv1 "backend/api/host/v1"
	"backend/internal/scripthost"
	"backend/pkg/config"
	"backend/pkg/log"
	grpcserver "backend/pkg/server/grpc"
	"go.uber.org/zap"
)

// OmniTask Host：Go 脚本宿主，以子进程 + stdio JSON 运行平台脚本。
func main() {
	var envConf = flag.String("conf", "config.yaml", "config path, eg: -conf ./config.yaml")
	flag.Parse()
	conf := config.NewConfig(*envConf)
	logger := log.NewLog(conf)

	host, err := scripthost.NewHost(conf, logger)
	if err != nil {
		panic(err)
	}

	srv := grpcserver.NewServer(
		logger,
		grpcserver.WithServerHost(conf.GetString("host.listen")),
		grpcserver.WithServerPort(conf.GetInt("host.port")),
	)
	hostv1.RegisterHostServiceServer(srv.Server, host)

	logger.Info("Host start",
		zap.String("addr", fmt.Sprintf("%s:%d", conf.GetString("host.listen"), conf.GetInt("host.port"))),
		zap.String("scripts_dir", conf.GetString("host.scripts_dir")),
	)
	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}
}
