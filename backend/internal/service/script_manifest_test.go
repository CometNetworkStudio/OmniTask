package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindManifestRequiresOmnitaskJson(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "script.json"), []byte(`{"id":"old"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := findManifest(dir); err == nil {
		t.Fatal("script.json should no longer be accepted")
	}
	target := filepath.Join(dir, "omnitask.json")
	if err := os.WriteFile(target, []byte(`{"id":"new"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("unexpected manifest path: %s", got)
	}
}
