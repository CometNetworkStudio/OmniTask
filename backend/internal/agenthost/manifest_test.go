package agenthost

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManifest(t *testing.T) {
	base := Manifest{
		Protocol: 2, ID: "demo", Name: "Demo", Runtime: "python3", Entry: "main.py",
		Actions: []Action{{ID: "run", Name: "Run"}},
	}
	if err := ValidateManifest(&base); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}

	dup := base
	dup.Actions = []Action{{ID: "run"}, {ID: "run"}}
	if err := ValidateManifest(&dup); err == nil {
		t.Fatal("duplicate action accepted")
	}

	noAction := base
	noAction.Actions = nil
	if err := ValidateManifest(&noAction); err == nil {
		t.Fatal("protocol v2 without actions accepted")
	}

	badDiscovery := base
	badDiscovery.Discovery = &Discovery{Actions: []string{"missing"}}
	if err := ValidateManifest(&badDiscovery); err == nil {
		t.Fatal("discovery referencing unknown action accepted")
	}
}

func TestLoadScriptsSkipsInvalidAndHandlesBOM(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good")
	bad := filepath.Join(dir, "bad")
	if err := os.MkdirAll(good, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "\xEF\xBB\xBF" + `{"protocol":2,"id":"good","name":"Good","runtime":"python3","entry":"main.py","actions":[{"id":"run","name":"Run"}]}`
	if err := os.WriteFile(filepath.Join(good, "omnitask.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "omnitask.json"), []byte(`{"id":"bad"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	scripts, err := LoadScripts(dir)
	if err != nil {
		t.Fatalf("load scripts: %v", err)
	}
	if len(scripts) != 1 || scripts[0].Manifest.ID != "good" {
		t.Fatalf("unexpected scripts: %+v", scripts)
	}
	if scripts[0].ManifestHash == "" || scripts[0].ManifestJSON == "" {
		t.Fatal("manifest hash/json not populated")
	}
}
