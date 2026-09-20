package agenthost

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Field struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Default any    `json:"default"`
}

type Action struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Manifest 对应脚本目录下的 script.json。
type Manifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Runtime     string   `json:"runtime"`
	Entry       string   `json:"entry"`
	Args        []string `json:"args"`
	Requires    []string `json:"requires"`
	Actions     []Action `json:"actions"`
	Credentials []Field  `json:"credentials"`
	Params      []Field  `json:"params"`
}

type Script struct {
	Manifest Manifest
	Dir      string
}

// LoadScripts 扫描 scripts_dir 下每个子目录的 script.json。
func LoadScripts(dir string) ([]Script, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Script
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name(), "script.json")
		buf, err := os.ReadFile(path)
		if err != nil {
			continue // 非脚本目录
		}
		var m Manifest
		if err := json.Unmarshal(buf, &m); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if m.ID == "" || m.Entry == "" {
			continue
		}
		out = append(out, Script{Manifest: m, Dir: filepath.Dir(path)})
	}
	return out, nil
}
