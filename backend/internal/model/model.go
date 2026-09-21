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

type Credential struct {
	ID        string `gorm:"primaryKey;size:32"`
	UserID    string `gorm:"uniqueIndex:uk_credential_user_platform;size:32;not null"`
	Platform  string `gorm:"uniqueIndex:uk_credential_user_platform;size:32;not null"`
	Cipher    []byte `gorm:"type:blob;not null"`
	Nonce     []byte `gorm:"type:blob;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Task struct {
	ID         string `gorm:"primaryKey;size:32"`
	UserID     string `gorm:"index;size:32;not null"`
	Platform   string `gorm:"size:32;not null"`
	Action     string `gorm:"size:64;not null"`
	Params     string `gorm:"type:json"`
	Status     string `gorm:"index;size:16;not null;default:pending"`
	Progress   int    `gorm:"not null;default:0"`
	Result     string `gorm:"type:json"`
	Error      string `gorm:"type:text"`
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TaskEvent struct {
	ID        string    `gorm:"primaryKey;size:32"`
	TaskID    string    `gorm:"index;size:32;not null"`
	Type      string    `gorm:"size:32;not null"`
	Percent   int       `gorm:"not null;default:0"`
	Message   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
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
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"size:128;not null;default:''"`
	Version     string `gorm:"size:32;not null;default:''"`
	SourceURL   string `gorm:"size:512;not null;default:''"`
	Manifest    string `gorm:"type:json"`
	Enabled     bool   `gorm:"not null;default:true"`
	Status      string `gorm:"size:32;not null;default:installed"`
	InstalledAt time.Time
	UpdatedAt   time.Time
}

// PlatformCache 缓存某用户在平台的“任务/条目列表”（原始字段存 JSON，前端按 manifest schema 渲染）。
type PlatformCache struct {
	ID          string `gorm:"primaryKey;size:32"`
	UserID      string `gorm:"uniqueIndex:uk_platform_cache;size:32;not null"`
	Platform    string `gorm:"uniqueIndex:uk_platform_cache;size:32;not null"`
	Items       string `gorm:"type:json"`
	RefreshedAt time.Time
	UpdatedAt   time.Time
}
