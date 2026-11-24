package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/crypto"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func TestMaskProxy(t *testing.T) {
	if got := maskProxy(""); got != "" {
		t.Fatalf("empty proxy: %q", got)
	}
	got := maskProxy("http://user:pass@127.0.0.1:8080")
	if got != "http://***@127.0.0.1:8080" {
		t.Fatalf("unexpected hint: %q", got)
	}
	if got := maskProxy("bad-value"); got != "已配置" {
		t.Fatalf("unexpected fallback: %q", got)
	}
}

func newTestCredentialService(t *testing.T) (*CredentialService, *repository.CredentialRepository, *repository.CredentialAccessLogRepository, *repository.TaskRepository) {
	t.Helper()
	conf := viper.New()
	conf.Set("log.log_level", "error")
	conf.Set("log.mode", "console")
	conf.Set("log.encoding", "console")
	logger := log.NewLog(conf)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Credential{}, &model.CredentialAccessLog{}, &model.PlatformConfig{}, &model.Task{}, &model.PlatformCache{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRepository(logger, db)
	credRepo := repository.NewCredentialRepository(repo)
	auditRepo := repository.NewCredentialAccessLogRepository(repo)
	taskRepo := repository.NewTaskRepository(repo)
	cacheRepo := repository.NewPlatformCacheRepository(repo)
	cfgSvc := NewPlatformConfigService(repository.NewPlatformConfigRepository(repo), repository.NewScriptRepository(repo), logger)
	keys, err := crypto.NewKeyring(map[int][]byte{1: []byte("0123456789abcdef0123456789abcdef")}, 1)
	if err != nil {
		t.Fatal(err)
	}
	return NewCredentialService(credRepo, auditRepo, taskRepo, cacheRepo, cfgSvc, keys, sid.NewSid(), logger, conf), credRepo, auditRepo, taskRepo
}

func TestCredentialLifecycleAuditAndPreview(t *testing.T) {
	svc, credRepo, auditRepo, taskRepo := newTestCredentialService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, "u1", &v1.CredentialCreateRequest{
		Platform: "welearn",
		Data:     map[string]string{"username": "18674271706", "password": "s3cret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.IsDefault {
		t.Fatal("first credential should be default")
	}
	if created.Preview != "186****1706" {
		t.Fatalf("unexpected preview: %q", created.Preview)
	}

	// 上限：默认 3
	for i := 0; i < 2; i++ {
		if _, err := svc.Create(ctx, "u1", &v1.CredentialCreateRequest{
			Platform: "welearn", Data: map[string]string{"username": "x", "password": "y"},
		}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Create(ctx, "u1", &v1.CredentialCreateRequest{
		Platform: "welearn", Data: map[string]string{"username": "x", "password": "y"},
	}); !errors.Is(err, v1.ErrCredentialLimit) {
		t.Fatalf("expected limit error, got %v", err)
	}

	creds, err := svc.Decrypted(ctx, "u1", created.ID, "welearn", CredentialAccess{Purpose: "task", RefID: "t1"})
	if err != nil || creds["password"] != "s3cret" {
		t.Fatalf("Decrypted: %v %v", creds, err)
	}
	used, err := credRepo.GetByID(ctx, "u1", created.ID)
	if err != nil || used.LastUsedAt == nil {
		t.Fatalf("last used should be recorded: %+v err=%v", used, err)
	}
	if used.KeyVersion != 1 {
		t.Fatalf("unexpected key version: %d", used.KeyVersion)
	}

	logs, err := auditRepo.ListByUser(ctx, "u1", 10)
	if err != nil || len(logs) != 1 || logs[0].CredentialID != created.ID || logs[0].RefID != "t1" {
		t.Fatalf("unexpected audit: %+v err=%v", logs, err)
	}

	stored, err := credRepo.GetByID(ctx, "u1", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stored.Cipher, []byte("s3cret")) || bytes.Contains(stored.Nonce, []byte("s3cret")) {
		t.Fatal("plaintext leaked into stored cipher/nonce")
	}

	// 有关联任务时归档而非删除
	if err := taskRepo.Create(ctx, &model.Task{ID: "t1", UserID: "u1", Platform: "welearn", Action: "study", CredentialID: created.ID, Status: "pending"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Archive(ctx, "u1", created.ID); err != nil {
		t.Fatal(err)
	}
	archived, err := credRepo.GetByID(ctx, "u1", created.ID)
	if err != nil || archived.ArchivedAt == nil {
		t.Fatalf("credential should be archived, got %+v err=%v", archived, err)
	}

	// 删除全部关联任务后，归档凭据被真正清除
	if _, err := taskRepo.Delete(ctx, "u1", "t1"); err != nil {
		t.Fatal(err)
	}
	svc.PurgeArchived(ctx, "u1", "welearn")
	if _, err := credRepo.GetByID(ctx, "u1", created.ID); err == nil {
		t.Fatal("archived credential should be purged after tasks removed")
	}
}
