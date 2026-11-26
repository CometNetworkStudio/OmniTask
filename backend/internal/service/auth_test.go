package service

import (
	"context"
	"errors"
	"testing"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func newTestAuthService(t *testing.T, configure func(*viper.Viper)) (*AuthService, *SessionService, *repository.UserRepository) {
	t.Helper()
	conf := viper.New()
	conf.Set("log.log_level", "error")
	conf.Set("log.mode", "console")
	conf.Set("log.encoding", "console")
	conf.Set("security.registration", "open")
	conf.Set("security.allow_first_user_admin", true)
	conf.Set("security.session_ttl_hours", 168)
	conf.Set("security.login_max_failures", 5)
	conf.Set("security.login_window_minutes", 15)
	if configure != nil {
		configure(conf)
	}
	logger := log.NewLog(conf)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Session{}, &model.LoginAttempt{}); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewRepository(logger, db)
	userRepo := repository.NewUserRepository(repo)
	sessionRepo := repository.NewSessionRepository(repo)
	attemptRepo := repository.NewLoginAttemptRepository(repo)
	s := sid.NewSid()
	sessions := NewSessionService(sessionRepo, attemptRepo, userRepo, s, conf, logger)
	return NewAuthService(userRepo, sessions, s, conf, logger), sessions, userRepo
}

func TestRegisterInviteAndFirstAdmin(t *testing.T) {
	auth, sessions, _ := newTestAuthService(t, func(conf *viper.Viper) {
		conf.Set("security.registration", "invite")
		conf.Set("security.invite_code", "let-me-in")
	})
	ctx := context.Background()

	_, _, err := auth.Register(ctx, &v1.RegisterRequest{Username: "alice", Password: "password1", InviteCode: "wrong"}, "1.1.1.1", "test")
	if !errors.Is(err, v1.ErrInviteInvalid) {
		t.Fatalf("expected invite error, got %v", err)
	}
	resp, token, err := auth.Register(ctx, &v1.RegisterRequest{Username: "alice", Password: "password1", InviteCode: "let-me-in"}, "1.1.1.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if resp.User.Role != "admin" {
		t.Fatalf("first user should be admin, got %q", resp.User.Role)
	}
	user, session, err := sessions.Validate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "alice" || session.UserID != user.ID {
		t.Fatalf("unexpected session: %+v %+v", user, session)
	}
}

func TestLoginLockoutAndDisabledUser(t *testing.T) {
	auth, sessions, users := newTestAuthService(t, func(conf *viper.Viper) {
		conf.Set("security.login_max_failures", 3)
		conf.Set("security.login_window_minutes", 15)
	})
	ctx := context.Background()

	if _, _, err := auth.Register(ctx, &v1.RegisterRequest{Username: "bob", Password: "password1"}, "1.1.1.1", "test"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_, _, err := auth.Login(ctx, &v1.LoginRequest{Username: "bob", Password: "bad-password"}, "1.1.1.1", "test")
		if !errors.Is(err, v1.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected invalid credentials, got %v", i+1, err)
		}
	}
	if _, _, err := auth.Login(ctx, &v1.LoginRequest{Username: "bob", Password: "password1"}, "1.1.1.1", "test"); !errors.Is(err, v1.ErrTooManyRequests) {
		t.Fatalf("expected lockout, got %v", err)
	}
	sessions.RecordAttempt(ctx, "bob", "1.1.1.1", "test", true, "success")
	resp, token, err := auth.Login(ctx, &v1.LoginRequest{Username: "bob", Password: "password1"}, "1.1.1.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := users.SetEnabled(ctx, resp.User.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := sessions.Validate(ctx, token); !errors.Is(err, v1.ErrUnauthorized) {
		t.Fatalf("disabled user session should be invalid, got %v", err)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	auth, sessions, _ := newTestAuthService(t, nil)
	ctx := context.Background()
	_, token, err := auth.Register(ctx, &v1.RegisterRequest{Username: "carol", Password: "password1"}, "1.1.1.1", "test")
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := sessions.Validate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.Logout(ctx, session.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := sessions.Validate(ctx, token); !errors.Is(err, v1.ErrUnauthorized) {
		t.Fatalf("logged out session should be invalid, got %v", err)
	}
}
