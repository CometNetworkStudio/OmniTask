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
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Command []string `json:"command"` // CLI 桥用：该动作要执行的命令（可含 {占位符}）
}

// Discovery 声明该平台“任务列表/发现”的展示 schema（列表接口返回原始字段，前端按此渲染）。
type Discovery struct {
	Label   string   `json:"label"`
	Columns []Field  `json:"columns"`
	Actions []string `json:"actions"`
}

// Manifest 对应脚本目录下的 script.json。
type Manifest struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	Author        string     `json:"author"`
	Runtime       string     `json:"runtime"`
	Entry         string     `json:"entry"`
	Module        string     `json:"module"` // 与 entry 二选一：用 python -m module 启动
	Args          []string   `json:"args"`
	Requires      []string   `json:"requires"`
	ProgressRegex string     `json:"progress_regex"` // CLI 桥从输出提取进度的正则（可选）
	Actions       []Action   `json:"actions"`
	Credentials   []Field    `json:"credentials"`
	Params        []Field    `json:"params"`
	Discovery     *Discovery `json:"discovery"`
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
		if m.ID == "" || (m.Entry == "" && m.Module == "") {
			continue
		}
		out = append(out, Script{Manifest: m, Dir: filepath.Dir(path)})
	}
	return out, nil
}
