package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

const defaultConfigName = "config.yaml"

const defaultTemplate = `# OmniTask 配置（首次运行自动生成）
# mode: lite（默认；SQLite，无需 MySQL/Redis）| full（MySQL + Redis）
mode: lite

http:
  host: 127.0.0.1
  port: 8080
  swagger: false
  cors_origins: ""
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
grpc:
  host: 127.0.0.1
  port: 9000
agent:
  addr: 127.0.0.1:50051
  host: 127.0.0.1
  port: 50051
  scripts_dir: scripts
  python: python3
  pythonpath: sdk/python
  bank_addr: 127.0.0.1:9000
  task_timeout: 0s
security:
  registration: open
  jwt:
    key: ""
  master_key: ""
data:
  db:
    migrate_on_start: false
    user:
      driver: sqlite
      dsn: data/omnitask.db
  redis:
    addr: 127.0.0.1:6350
    password: ""
    db: 0
pagination:
  default: 100
  max: 1000
log:
  log_level: info
  mode: both
  encoding: console
  log_file_name: ./logs/server.log
  max_backups: 30
  max_age: 7
  max_size: 1024
  compress: true
`

func NewConfig(p string) *viper.Viper {
	envConf := os.Getenv("APP_CONF")
	if envConf == "" {
		envConf = p
	}
	fmt.Println("load conf file:", envConf)
	if envConf == defaultConfigName {
		ensureDefaultConfig(envConf)
	}
	conf := getConfig(envConf)
	postProcess(conf, envConf)
	return conf
}

func getConfig(path string) *viper.Viper {
	conf := viper.New()
	conf.SetConfigFile(path)
	if err := conf.ReadInConfig(); err != nil {
		panic(err)
	}
	return conf
}

func ensureDefaultConfig(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := os.WriteFile(path, []byte(defaultTemplate), 0o644); err != nil {
		panic(fmt.Errorf("write default config: %w", err))
	}
	fmt.Println("created default config:", path)
}

// postProcess 补齐 mode/密钥、lite 驱动、必要目录，并回写配置。
func postProcess(conf *viper.Viper, path string) {
	if conf.GetString("mode") == "" {
		conf.Set("mode", "lite")
	}
	if conf.GetString("security.jwt.key") == "" {
		conf.Set("security.jwt.key", randomHex(32))
	}
	if conf.GetString("security.master_key") == "" && os.Getenv("OMNITASK_MASTER_KEY") == "" {
		conf.Set("security.master_key", randomHex(32))
	}
	if conf.GetString("mode") == "lite" {
		conf.Set("data.db.user.driver", "sqlite")
		if conf.GetString("data.db.user.dsn") == "" {
			conf.Set("data.db.user.dsn", "data/omnitask.db")
		}
	}
	dirs := []string{"data", "logs", conf.GetString("agent.scripts_dir")}
	for _, dir := range dirs {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}
	if err := conf.WriteConfigAs(path); err != nil {
		fmt.Println("warn: persist config failed:", err)
	}
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
