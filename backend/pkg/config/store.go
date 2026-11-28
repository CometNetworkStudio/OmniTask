package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// FieldType 是管理端配置编辑支持的字段类型。
type FieldType string

const (
	FieldString FieldType = "string"
	FieldInt    FieldType = "int"
	FieldBool   FieldType = "bool"
	FieldSelect FieldType = "select"
)

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field 是返回给前端的可编辑配置项。
type Field struct {
	Key     string    `json:"key"`
	Label   string    `json:"label"`
	Group   string    `json:"group"`
	Type    FieldType `json:"type"`
	Value   any       `json:"value"`
	Default any       `json:"default"`
	Min     int       `json:"min,omitempty"`
	Max     int       `json:"max,omitempty"`
	Options []Option  `json:"options,omitempty"`
	Help    string    `json:"help,omitempty"`
	Effect  string    `json:"effect,omitempty"`
}

type RestartItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Note  string `json:"note,omitempty"`
}

// Snapshot 是管理端配置页的完整数据，不包含任何密钥。
type Snapshot struct {
	File         string        `json:"file"`
	OverrideFile string        `json:"override_file"`
	Fields       []Field       `json:"fields"`
	RestartItems []RestartItem `json:"restart_items"`
	Warnings     []string      `json:"warnings,omitempty"`
}

type fieldSpec struct {
	Key     string
	Label   string
	Group   string
	Type    FieldType
	Default any
	Min     int
	Max     int
	Options []Option
	Help    string
	Effect  string
}

// editableFields 是管理端唯一允许修改的配置白名单。密钥、数据库、Host 等不在其中。
var editableFields = []fieldSpec{
	{
		Key: "security.registration", Label: "注册方式", Group: "注册与登录", Type: FieldSelect,
		Default: "open", Effect: "立即生效",
		Options: []Option{{Value: "open", Label: "开放注册"}, {Value: "invite", Label: "邀请码注册"}, {Value: "closed", Label: "关闭注册"}},
		Help:    "开放注册适合小范围信任环境；公开部署建议使用邀请码或关闭注册。",
	},
	{
		Key: "security.invite_code", Label: "邀请码", Group: "注册与登录", Type: FieldString,
		Default: "", Effect: "立即生效", Help: "仅在邀请码注册模式下使用。留空会导致所有注册被拒绝。",
	},
	{
		Key: "security.allow_first_user_admin", Label: "首个用户自动成为管理员", Group: "注册与登录", Type: FieldBool,
		Default: true, Effect: "仅对空用户库生效", Help: "官方托管建议关闭，并通过环境变量预创建管理员。",
	},
	{
		Key: "security.auth_rate_limit_per_minute", Label: "认证接口每 IP 每分钟上限", Group: "注册与登录", Type: FieldInt,
		Default: 20, Min: 0, Max: 1000, Effect: "立即生效", Help: "0 表示关闭。覆盖登录和注册接口。",
	},
	{
		Key: "security.login_max_failures", Label: "登录失败锁定次数", Group: "注册与登录", Type: FieldInt,
		Default: 5, Min: 0, Max: 100, Effect: "立即生效", Help: "同一用户名在窗口内失败达到该次数后暂时拒绝登录，0 表示关闭。",
	},
	{
		Key: "security.login_window_minutes", Label: "登录失败统计窗口（分钟）", Group: "注册与登录", Type: FieldInt,
		Default: 15, Min: 1, Max: 1440, Effect: "立即生效",
	},
	{
		Key: "security.session_ttl_hours", Label: "会话有效期（小时）", Group: "会话", Type: FieldInt,
		Default: 168, Min: 1, Max: 8760, Effect: "仅新登录生效", Help: "修改不会延长或缩短已经建立的会话。",
	},
	{
		Key: "security.cookie_secure", Label: "Cookie Secure 策略", Group: "会话", Type: FieldSelect,
		Default: "auto", Effect: "立即生效",
		Options: []Option{{Value: "auto", Label: "自动判断"}, {Value: "true", Label: "始终 Secure"}, {Value: "false", Label: "不强制 Secure"}},
		Help:    "自动判断依据 HTTPS 和 X-Forwarded-Proto；生产环境应保持自动或始终 Secure。",
	},
	{
		Key: "security.cookie_same_site", Label: "Cookie SameSite", Group: "会话", Type: FieldSelect,
		Default: "lax", Effect: "立即生效",
		Options: []Option{{Value: "lax", Label: "Lax"}, {Value: "strict", Label: "Strict"}, {Value: "none", Label: "None"}},
		Help:    "None 需要 HTTPS 和 Secure，通常只用于跨站前端。",
	},
	{
		Key: "task.max_active_per_user", Label: "单用户活跃任务上限", Group: "任务配额", Type: FieldInt,
		Default: 20, Min: 0, Max: 10000, Effect: "立即生效", Help: "0 表示不限。",
	},
	{
		Key: "task.daily_tasks_per_user", Label: "单用户每日任务上限", Group: "任务配额", Type: FieldInt,
		Default: 0, Min: 0, Max: 100000, Effect: "立即生效", Help: "0 表示不限。",
	},
	{
		Key: "task.max_pending_per_user", Label: "单用户待处理任务上限", Group: "任务配额", Type: FieldInt,
		Default: 0, Min: 0, Max: 100000, Effect: "立即生效", Help: "0 表示不限。",
	},
	{
		Key: "task.max_pending", Label: "全局待处理任务上限", Group: "任务配额", Type: FieldInt,
		Default: 500, Min: 0, Max: 1000000, Effect: "立即生效", Help: "0 表示不限。",
	},
	{
		Key: "discovery.min_interval", Label: "平台列表刷新最小间隔（秒）", Group: "发现", Type: FieldInt,
		Default: 60, Min: 0, Max: 86400, Effect: "立即生效", Help: "0 表示不限制。",
	},
}

// Store 管理主配置、覆盖文件和运行中的 viper 实例。
type Store struct {
	conf         *viper.Viper
	mainPath     string
	overridePath string
	mu           sync.Mutex
}

func NewStore(conf *viper.Viper) *Store {
	path := activePath
	if path == "" {
		path = "config.yaml"
	}
	return &Store{conf: conf, mainPath: path, overridePath: overridePathFor(path)}
}

func (s *Store) Snapshot() *Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *Store) snapshotLocked() *Snapshot {
	fields := make([]Field, 0, len(editableFields))
	warnings := make([]string, 0)
	for _, spec := range editableFields {
		value := s.read(spec)
		fields = append(fields, Field{
			Key: spec.Key, Label: spec.Label, Group: spec.Group, Type: spec.Type,
			Value: value, Default: spec.Default, Min: spec.Min, Max: spec.Max,
			Options: spec.Options, Help: spec.Help, Effect: spec.Effect,
		})
		if spec.Key == "security.registration" && value == "invite" {
			if strings.TrimSpace(s.conf.GetString("security.invite_code")) == "" {
				warnings = append(warnings, "当前是邀请码注册，但邀请码为空，所有注册都会被拒绝。")
			}
		}
	}
	return &Snapshot{
		File:         s.mainPath,
		OverrideFile: s.overridePath,
		Fields:       fields,
		RestartItems: restartItems,
		Warnings:     warnings,
	}
}

var restartItems = []RestartItem{
	{Key: "env", Label: "运行环境", Note: "prod 会关闭 Swagger 和 SQL 明细日志"},
	{Key: "mode", Label: "lite / full 模式"},
	{Key: "http.host", Label: "HTTP 监听地址"},
	{Key: "http.port", Label: "HTTP 监听端口"},
	{Key: "http.trusted_proxies", Label: "可信反向代理"},
	{Key: "http.cors_origins", Label: "跨域来源"},
	{Key: "http.tls.*", Label: "TLS 证书"},
	{Key: "data.db.*", Label: "数据库连接"},
	{Key: "data.redis.*", Label: "Redis 连接"},
	{Key: "host.*", Label: "Host 与脚本运行参数"},
	{Key: "log.*", Label: "日志参数"},
	{Key: "security.master_key*", Label: "凭据主密钥", Note: "请通过环境变量或文件修改"},
}

// Update 校验并写入覆盖文件，然后把可热更新的值应用到运行中的配置。
func (s *Store) Update(patch map[string]any) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(patch) == 0 {
		return s.snapshotLocked(), nil
	}
	specByKey := make(map[string]fieldSpec, len(editableFields))
	for _, spec := range editableFields {
		specByKey[spec.Key] = spec
	}
	normalized := make(map[string]any, len(patch))
	for key, raw := range patch {
		spec, ok := specByKey[key]
		if !ok {
			return nil, fmt.Errorf("不支持的配置项: %s", key)
		}
		value, err := normalize(spec, raw)
		if err != nil {
			return nil, err
		}
		normalized[key] = value
	}

	override := viper.New()
	override.SetConfigFile(s.overridePath)
	if _, err := os.Stat(s.overridePath); err == nil {
		if err := override.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取覆盖配置失败: %w", err)
		}
	}
	for key, value := range normalized {
		override.Set(key, value)
	}
	if err := os.MkdirAll(filepath.Dir(s.overridePath), 0o755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}
	if err := override.WriteConfigAs(s.overridePath); err != nil {
		return nil, fmt.Errorf("写入覆盖配置失败: %w", err)
	}
	_ = os.Chmod(s.overridePath, 0o600)

	for key, value := range normalized {
		s.conf.Set(key, value)
	}
	return s.snapshotLocked(), nil
}

// Reload 重新读取主配置和覆盖文件，仅刷新可编辑字段，用于手动改文件后立即应用。
func (s *Store) Reload() (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fresh := viper.New()
	fresh.SetConfigFile(s.mainPath)
	if err := fresh.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取主配置失败: %w", err)
	}
	mergeOverride(fresh, s.mainPath)
	for _, spec := range editableFields {
		s.conf.Set(spec.Key, readFrom(fresh, spec))
	}
	return s.snapshotLocked(), nil
}

func (s *Store) read(spec fieldSpec) any {
	return readFrom(s.conf, spec)
}

func readFrom(conf *viper.Viper, spec fieldSpec) any {
	if !conf.IsSet(spec.Key) {
		return spec.Default
	}
	switch spec.Type {
	case FieldInt:
		return conf.GetInt(spec.Key)
	case FieldBool:
		return conf.GetBool(spec.Key)
	default:
		return conf.GetString(spec.Key)
	}
}

func normalize(spec fieldSpec, raw any) (any, error) {
	switch spec.Type {
	case FieldInt:
		var value int
		switch v := raw.(type) {
		case float64:
			if math.Trunc(v) != v {
				return nil, fmt.Errorf("%s 必须是整数", spec.Label)
			}
			value = int(v)
		case int:
			value = v
		case json.Number:
			n, err := v.Int64()
			if err != nil {
				return nil, fmt.Errorf("%s 必须是整数", spec.Label)
			}
			value = int(n)
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("%s 必须是整数", spec.Label)
			}
			value = n
		default:
			return nil, fmt.Errorf("%s 必须是整数", spec.Label)
		}
		if value < spec.Min || value > spec.Max {
			return nil, fmt.Errorf("%s 必须在 %d 到 %d 之间", spec.Label, spec.Min, spec.Max)
		}
		return value, nil
	case FieldBool:
		switch v := raw.(type) {
		case bool:
			return v, nil
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(v))
			if err != nil {
				return nil, fmt.Errorf("%s 必须是布尔值", spec.Label)
			}
			return parsed, nil
		default:
			return nil, fmt.Errorf("%s 必须是布尔值", spec.Label)
		}
	case FieldSelect:
		value := strings.TrimSpace(fmt.Sprint(raw))
		for _, option := range spec.Options {
			if option.Value == value {
				return value, nil
			}
		}
		return nil, fmt.Errorf("%s 的取值不合法", spec.Label)
	case FieldString:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("%s 必须是文本", spec.Label)
		}
		value = strings.TrimSpace(value)
		if len(value) > 512 {
			return nil, fmt.Errorf("%s 过长", spec.Label)
		}
		return value, nil
	default:
		return nil, errors.New("未知字段类型")
	}
}
