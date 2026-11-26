package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "config.yaml")
	body := `security:
  registration: open
  invite_code: ""
  allow_first_user_admin: true
  auth_rate_limit_per_minute: 20
  login_max_failures: 5
  login_window_minutes: 15
  session_ttl_hours: 168
  cookie_secure: auto
  cookie_same_site: lax
task:
  max_active_per_user: 20
  daily_tasks_per_user: 0
  max_pending_per_user: 0
  max_pending: 500
discovery:
  min_interval: 60
`
	if err := os.WriteFile(mainPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	activePath = mainPath
	conf := viper.New()
	conf.SetConfigFile(mainPath)
	if err := conf.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	return NewStore(conf), mainPath
}

func TestStoreUpdateWritesOverrideAndApplies(t *testing.T) {
	store, mainPath := newTestStore(t)
	snapshot := store.Snapshot()
	if snapshot.File != mainPath {
		t.Fatalf("unexpected file: %s", snapshot.File)
	}
	if snapshot.OverrideFile != strings.TrimSuffix(mainPath, ".yaml")+".overrides.yaml" {
		t.Fatalf("unexpected override path: %s", snapshot.OverrideFile)
	}

	updated, err := store.Update(map[string]any{
		"security.registration":     "invite",
		"security.invite_code":      "secret-code",
		"task.daily_tasks_per_user": float64(20),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := updatedValue(updated, "security.registration"); got != "invite" {
		t.Fatalf("registration not applied: %v", got)
	}
	if store.conf.GetString("security.registration") != "invite" {
		t.Fatal("live config not updated")
	}
	if store.conf.GetString("security.invite_code") != "secret-code" {
		t.Fatal("invite code not updated")
	}
	if store.conf.GetInt("task.daily_tasks_per_user") != 20 {
		t.Fatal("daily quota not updated")
	}
	if _, err := os.Stat(snapshot.OverrideFile); err != nil {
		t.Fatalf("override file missing: %v", err)
	}
	// 主配置文件应保持原样，不被可视化编辑重写。
	mainBody, _ := os.ReadFile(mainPath)
	if !strings.Contains(string(mainBody), "registration: open") {
		t.Fatal("main config should not be rewritten")
	}
}

func TestStoreValidation(t *testing.T) {
	store, _ := newTestStore(t)
	if _, err := store.Update(map[string]any{"security.master_key": "x"}); err == nil {
		t.Fatal("secret key must not be editable")
	}
	if _, err := store.Update(map[string]any{"security.registration": "nope"}); err == nil {
		t.Fatal("invalid enum should fail")
	}
	if _, err := store.Update(map[string]any{"security.login_max_failures": float64(1000)}); err == nil {
		t.Fatal("out of range int should fail")
	}
	if _, err := store.Update(map[string]any{"security.login_max_failures": 3.5}); err == nil {
		t.Fatal("fractional int should fail")
	}
}

func TestStoreReload(t *testing.T) {
	store, mainPath := newTestStore(t)
	if _, err := store.Update(map[string]any{"security.registration": "invite"}); err != nil {
		t.Fatal(err)
	}
	if store.conf.GetString("security.registration") != "invite" {
		t.Fatal("expected invite")
	}
	if err := os.Remove(store.overridePath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte("security:\n  registration: closed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reload(); err != nil {
		t.Fatal(err)
	}
	if store.conf.GetString("security.registration") != "closed" {
		t.Fatalf("reload did not pick main config: %q", store.conf.GetString("security.registration"))
	}
}

func updatedValue(snapshot *Snapshot, key string) any {
	for _, field := range snapshot.Fields {
		if field.Key == key {
			return field.Value
		}
	}
	return nil
}
