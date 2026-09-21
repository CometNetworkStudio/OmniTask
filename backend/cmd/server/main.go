package main

import (
	"context"
	"flag"
	"fmt"

	"backend/cmd/server/wire"
	"backend/internal/migration"
	"backend/pkg/config"
	"backend/pkg/log"
	"go.uber.org/zap"
)

// @title           OmniTask API
// @version         1.0.0
// @description     OmniTask 万务通 后端 API
// @termsOfService  http://swagger.io/terms/
// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
// @host      localhost:8080
// @securityDefinitions.apiKey Bearer
// @in header
// @name Authorization
// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	var envConf = flag.String("conf", "config.yaml", "config path, eg: -conf ./config.yaml")
	var migrateOnly = flag.Bool("migrate", false, "仅执行数据库迁移后退出（full 模式）")
	flag.Parse()
	conf := config.NewConfig(*envConf)

	logger := log.NewLog(conf)

	if *migrateOnly {
		if err := migration.Up(conf); err != nil {
			logger.Sugar().Fatalf("migrate failed: %v", err)
		}
		logger.Info("migrate done")
		return
	}
	if conf.GetBool("data.db.migrate_on_start") {
		if err := migration.Up(conf); err != nil {
			logger.Sugar().Fatalf("migrate on start failed: %v", err)
		}
	}

	app, cleanup, err := wire.NewWire(conf, logger)
	defer cleanup()
	if err != nil {
		panic(err)
	}
	scheme := "http"
	if conf.GetBool("http.tls.enabled") {
		scheme = "https"
	}
	logger.Info("server start", zap.String("host", fmt.Sprintf("%s://%s:%d", scheme, conf.GetString("http.host"), conf.GetInt("http.port"))))
	logger.Info("docs addr", zap.String("addr", fmt.Sprintf("%s://%s:%d/swagger/index.html", scheme, conf.GetString("http.host"), conf.GetInt("http.port"))))
	if err = app.Run(context.Background()); err != nil {
		panic(err)
	}
}
