package model

import "time"

// 本文件冻结 OmniTask 的持久化 schema（M0.5）。
// 约定：
//   - 主键统一为 sonyflake base62 字符串（由 service 层的 sid 生成）。
//   - JSON 字段以 string 存储，列类型标注 json；跨 sqlite(dev)/mysql(prod) 均可。
//   - 凭据密文只存 cipher+nonce，禁止任何明文列。

type User struct {
	ID           string `gorm:"primaryKey;size:32"`
	Username     string `gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	Role         string `gorm:"size:32;not null;default:user"`
	Enabled      bool   `gorm:"not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserPreference 保存用户级界面偏好（当前用于侧边栏置顶平台）。
type UserPreference struct {
	UserID          string `gorm:"primaryKey;size:32"`
	PinnedPlatforms string `gorm:"type:text;not null"` // JSON 数组
	UpdatedAt       time.Time
}

// Credential 是用户在某平台的一份凭据，同一用户同一平台可有多份（受平台配置上限约束）。
// Label/Preview 为非敏感的展示信息（Preview 由后端脱敏生成），Cipher/Nonce 为密文。
type Credential struct {
	ID         string     `gorm:"primaryKey;size:32"`
	UserID     string     `gorm:"index:idx_credential_user_platform;size:32;not null"`
	Platform   string     `gorm:"index:idx_credential_user_platform;size:32;not null"`
	Label      string     `gorm:"size:64"`
	Preview    string     `gorm:"size:128"`
	ProxyHint  string     `gorm:"size:255"`
	IsDefault  bool       `gorm:"not null;default:false"`
	ArchivedAt *time.Time // 归档后从列表/选择器隐藏，历史任务删净后真正清除
	KeyVersion int        `gorm:"not null;default:1"` // 加密所用主密钥版本，支持轮换
	LastUsedAt *time.Time // 最近一次任务/发现解密时间，仅用于展示
	Cipher     []byte     `gorm:"type:blob"`
	Nonce      []byte     `gorm:"type:blob"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PlatformConfig 是平台级配置（启动时按已安装脚本写入默认值）。
type PlatformConfig struct {
	PlatformID            string `gorm:"primaryKey;size:64"`
	MaxCredentials        int    `gorm:"not null;default:3"`
	DefaultCredentialType string `gorm:"size:16"`
	PreviewFields         string `gorm:"type:json"` // 用于展示的标识字段 key 列表
	TokenHeadLen          int    `gorm:"not null;default:6"`
	TokenTailLen          int    `gorm:"not null;default:4"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// CredentialAccessLog 记录每次凭据解密的元数据（绝不含明文），
// 用于内部约束与对用户透明的访问审计。
type CredentialAccessLog struct {
	ID           string    `gorm:"primaryKey;size:32"`
	UserID       string    `gorm:"index:idx_cal_user_created;size:32;not null"`
	Platform     string    `gorm:"size:32;not null"`
	CredentialID string    `gorm:"size:32"`
	Purpose      string    `gorm:"size:16;not null"` // task | discover
	RefID        string    `gorm:"size:32"`          // task_id，discover 为空
	CreatedAt    time.Time `gorm:"index:idx_cal_user_created"`
}

type Task struct {
	ID           string `gorm:"primaryKey;size:32"`
	UserID       string `gorm:"index;size:32;not null"`
	Platform     string `gorm:"size:32;not null"`
	Action       string `gorm:"size:64;not null"`
	CredentialID string `gorm:"index;size:32"`
	Params       string `gorm:"type:json"`
	Status       string `gorm:"index;size:16;not null;default:pending"`
	Progress     int    `gorm:"not null;default:0"`
	Result       string `gorm:"type:json"`
	Error        string `gorm:"type:text"`
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskEvent struct {
	ID         string    `gorm:"primaryKey;size:32"`
	TaskID     string    `gorm:"index;size:32;not null"`
	Type       string    `gorm:"size:32;not null"`                  // 任务状态：running/succeeded/failed...
	Kind       string    `gorm:"size:32;not null;default:progress"` // 展示类型：progress/log/step/metric/started
	Percent    int       `gorm:"not null;default:0"`
	Message    string    `gorm:"type:text"`
	Level      string    `gorm:"size:16"`
	Step       string    `gorm:"size:64"`
	Label      string    `gorm:"size:128"`
	StepStatus string    `gorm:"size:16"`
	Current    int       `gorm:"not null;default:0"`
	Total      int       `gorm:"not null;default:0"`
	Unit       string    `gorm:"size:16"`
	Stage      string    `gorm:"size:64"`
	Key        string    `gorm:"size:64"`
	Value      string    `gorm:"size:255"`
	CreatedAt  time.Time `gorm:"index"`
}

type Question struct {
	ID          string  `gorm:"primaryKey;size:32"`
	Platform    string  `gorm:"uniqueIndex:uk_question_platform_key;size:32;not null"`
	QuestionKey string  `gorm:"uniqueIndex:uk_question_platform_key;size:128;not null"`
	Stem        string  `gorm:"type:text"`
	Options     string  `gorm:"type:json"`
	Answer      string  `gorm:"type:json"`
	Source      string  `gorm:"size:16;not null"`
	Confidence  float64 `gorm:"not null;default:0"`
	HitCount    int64   `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Script 是已安装的平台脚本注册表。
type Script struct {
	ID           string `gorm:"primaryKey;size:64"`
	Name         string `gorm:"size:128;not null;default:''"`
	Version      string `gorm:"size:32;not null;default:''"`
	SourceURL    string `gorm:"size:512;not null;default:''"`
	Ref          string `gorm:"size:128;not null;default:''"`
	CommitSHA    string `gorm:"size(64);not null;default:''"`
	ManifestHash string `gorm:"size:64;not null;default:''"`
	Manifest     string `gorm:"type:json"`
	Enabled      bool   `gorm:"not null;default:true"`
	Status       string `gorm:"size:32;not null;default:installed"`
	LastError    string `gorm:"type:text"`
	InstalledAt  time.Time
	UpdatedAt    time.Time
}

// ScriptInstallJob 把安装进度落库，服务重启后仍可查询。
type ScriptInstallJob struct {
	ID        string `gorm:"primaryKey;size:32"`
	Status    string `gorm:"size:16;not null;default:running"`
	Stage     string `gorm:"size:32;not null;default:queued"`
	Percent   int    `gorm:"not null;default:0"`
	Message   string `gorm:"type:text"`
	Error     string `gorm:"type:text"`
	ScriptID  string `gorm:"size:64"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PlatformCache 缓存某用户某凭据在平台的“任务/条目列表”（原始字段存 JSON，前端按 manifest schema 渲染）。
type PlatformCache struct {
	ID           string `gorm:"primaryKey;size:32"`
	UserID       string `gorm:"uniqueIndex:uk_platform_cache;size:32;not null"`
	Platform     string `gorm:"uniqueIndex:uk_platform_cache;size:32;not null"`
	CredentialID string `gorm:"uniqueIndex:uk_platform_cache;size:32;not null;default:''"`
	Items        string `gorm:"type:json"`
	RefreshedAt  time.Time
	UpdatedAt    time.Time
}
