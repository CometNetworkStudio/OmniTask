package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

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
  # 可信反向代理网段，逗号分隔；为空表示不信任任何代理（直接取连接 IP）。
  # 经 nginx 部署时填代理所在网段，否则登录限流会把所有用户算到代理 IP 上。
  trusted_proxies: ""
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
  global_concurrency: 4
  platform_concurrency: 2
  auto_install_deps: false
  pip_index_url: ""
  autostart: true
  bin: ""
  workspaces_dir: workspaces
  rate_dir: rate
  max_processes: 8
  keep_workspaces: false
  keep_failed_workspaces: true
  keep_versions: 3
security:
  registration: open
  jwt:
    key: ""
  # 当前主密钥（默认版本 1）。生产环境建议改用环境变量 OMNITASK_MASTER_KEY。
  master_key: ""
  # 轮换用：版本号 -> 密钥。非空时只读此映射，必须包含旧密钥版本。
  # 未显式指定 master_key_version 时，最大版本用于新写入。
  # 密钥建议加引号，避免 YAML 把纯数字串当成数值。
  master_keys: {}
  # 可选：显式指定新写入使用的密钥版本；留空或 0 表示取最大版本。
  # master_key_version: 2
  # true 时，凭据访问审计写不进去就不允许解密使用。
  credential_audit_fail_closed: false
  # 登录/注册接口的每 IP 每分钟请求上限，0 表示关闭。
  auth_rate_limit_per_minute: 20
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
task:
  max_active_per_user: 20
  max_pending: 500
discovery:
  min_interval: 60
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
	if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
		panic(fmt.Errorf("write default config: %w", err))
	}
	fmt.Println("created default config:", path)
}

// postProcess 补齐 mode/密钥、lite 驱动、必要目录；仅在生成新密钥时回写配置。
// 避免每次启动重写用户的 config.yaml（会丢失注释，也可能改写纯数字密钥）。
func postProcess(conf *viper.Viper, path string) {
	persist := false
	if conf.GetString("mode") == "" {
		conf.Set("mode", "lite")
	}
	if conf.GetString("security.jwt.key") == "" {
		conf.Set("security.jwt.key", randomHex(32))
		persist = true
	}
	if conf.GetString("security.master_key") == "" && os.Getenv("OMNITASK_MASTER_KEY") == "" {
		conf.Set("security.master_key", randomHex(32))
		persist = true
	}
	if conf.GetString("mode") == "lite" {
		conf.Set("data.db.user.driver", "sqlite")
		if conf.GetString("data.db.user.dsn") == "" {
			conf.Set("data.db.user.dsn", "data/omnitask.db")
		}
	}
	if !conf.IsSet("agent.autostart") {
		conf.Set("agent.autostart", conf.GetString("mode") == "lite")
	}
	defaults := map[string]any{
		"agent.workspaces_dir":                  "workspaces",
		"agent.rate_dir":                        "rate",
		"agent.max_processes":                   8,
		"agent.keep_workspaces":                 false,
		"agent.keep_failed_workspaces":          true,
		"agent.keep_versions":                   3,
		"task.max_active_per_user":              20,
		"task.max_pending":                      500,
		"http.trusted_proxies":                  "",
		"security.credential_audit_fail_closed": false,
		"security.auth_rate_limit_per_minute":   20,
	}
	for key, value := range defaults {
		if !conf.IsSet(key) {
			conf.Set(key, value)
		}
	}
	dirs := []string{"data", "logs", conf.GetString("agent.scripts_dir")}
	for _, dir := range dirs {
		if dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
	}
	if os.Getenv("OMNITASK_CONFIG_NO_WRITE") != "1" && persist {
		if err := conf.WriteConfigAs(path); err != nil {
			fmt.Println("warn: persist config failed:", err)
		}
		// 配置里含 JWT 与主密钥，尽量收紧权限；只读挂载失败可忽略。
		_ = os.Chmod(path, 0o600)
	}
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	s := hex.EncodeToString(buf)
	// 纯数字字符串会被 YAML 当成数值，回写配置时可能被改写；确保至少含一个字母。
	if n > 0 && strings.Trim(s, "0123456789") == "" {
		s = "a" + s[1:]
	}
	return s
}
