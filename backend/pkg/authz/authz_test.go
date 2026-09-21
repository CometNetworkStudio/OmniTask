package authz

import (
	"testing"

	"backend/pkg/log"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func TestEnforcerRBAC(t *testing.T) {
	conf := viper.New()
	conf.Set("log.log_level", "error")
	conf.Set("log.mode", "console")
	conf.Set("log.encoding", "console")
	logger := log.NewLog(conf)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewEnforcer(db, logger)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		sub, obj, act string
		want          bool
	}{
		{"admin", "/api/v1/users", "GET", true},
		{"admin", "/api/v1/scripts/install", "POST", true},
		{"user", "/api/v1/users", "GET", false},
		{"user", "/api/v1/scripts", "GET", false},
		{"user", "/api/v1/tasks", "GET", true},
		{"user", "/api/v1/tasks", "POST", true},
		{"user", "/api/v1/tasks/:id", "DELETE", true},
		{"user", "/api/v1/me", "GET", true},
	}
	for _, c := range cases {
		got, err := e.Enforce(c.sub, c.obj, c.act)
		if err != nil {
			t.Fatalf("enforce error: %v", err)
		}
		if got != c.want {
			t.Errorf("Enforce(%s,%s,%s)=%v want %v", c.sub, c.obj, c.act, got, c.want)
		}
	}
}
