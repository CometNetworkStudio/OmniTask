package scripthost

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyWorkspaceSkipsRuntimeAndCache(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	write := func(rel, body string) {
		path := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("main.py", "print('ok')")
	write("config.ini", "mode=source")
	write(".venv/bin/python", "binary")
	write("__pycache__/main.pyc", "cache")
	write("node_modules/pkg/index.js", "module")
	write("assets/data.txt", "asset")

	if err := copyWorkspace(src, dst); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"main.py", "config.ini", "assets/data.txt"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Fatalf("%s should be copied: %v", rel, err)
		}
	}
	for _, rel := range []string{".venv", "__pycache__", "node_modules", "main.pyc"} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err == nil {
			t.Fatalf("%s should be skipped", rel)
		}
	}
}

func TestBaseEnvDropsSecrets(t *testing.T) {
	t.Setenv("OMNITASK_MASTER_KEY", "super-secret")
	t.Setenv("OMNITASK_DB_DSN", "root:password@tcp(db)/omnitask")
	t.Setenv("PATH", "/usr/bin")
	for _, item := range baseEnv() {
		if strings.HasPrefix(item, "OMNITASK_MASTER_KEY=") || strings.HasPrefix(item, "OMNITASK_DB_DSN=") {
			t.Fatalf("sensitive variable leaked: %s", item)
		}
	}
}

func TestActiveVersionDir(t *testing.T) {
	root := t.TempDir()
	version := filepath.Join(root, "versions", "abc")
	if err := os.MkdirAll(version, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(version, "omnitask.json"), []byte(`{"id":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "active.json"), []byte(`{"active":"abc"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, key, ok := activeVersionDir(root)
	if !ok || key != "abc" || dir != version {
		t.Fatalf("unexpected active version: %q %q %v", dir, key, ok)
	}
}
