package service

import (
	"context"
	"testing"

	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func newTestUserService(t *testing.T) (*UserService, *repository.UserRepository) {
	t.Helper()
	conf := viper.New()
	conf.Set("pagination.default", 100)
	conf.Set("pagination.max", 1000)
	conf.Set("log.log_level", "error")
	conf.Set("log.mode", "console")
	conf.Set("log.encoding", "console")
	logger := log.NewLog(conf)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRepository(logger, db)
	userRepo := repository.NewUserRepository(repo)
	return NewUserService(userRepo, conf, logger), userRepo
}

func TestUserServiceLifecycle(t *testing.T) {
	svc, userRepo := newTestUserService(t)
	ctx := context.Background()

	if err := userRepo.Create(ctx, &model.User{
		ID: "u1", Username: "alice", PasswordHash: "x", Role: "user", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	me, err := svc.Me(ctx, "u1")
	if err != nil || me.Username != "alice" || !me.Enabled {
		t.Fatalf("Me: %+v err=%v", me, err)
	}

	list, total, err := svc.List(ctx, 1, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("List: total=%d len=%d err=%v", total, len(list), err)
	}

	if _, err := svc.UpdateRole(ctx, "u1", "admin"); err != nil {
		t.Fatal(err)
	}
	if me, _ := svc.Me(ctx, "u1"); me.Role != "admin" {
		t.Fatalf("role not updated: %s", me.Role)
	}

	if _, err := svc.SetEnabled(ctx, "u1", false); err != nil {
		t.Fatal(err)
	}
	if me, _ := svc.Me(ctx, "u1"); me.Enabled {
		t.Fatal("enabled not updated")
	}
}
