package agenthost

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ProtocolVersion  = 2
	ManifestFileName = "omnitask.json"
)

type Option struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// OptionList 兼容 ["a","b"] 与 [{"label":"A","value":"a"}] 两种写法。
type OptionList []Option

func (o *OptionList) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(OptionList, 0, len(raw))
	for _, item := range raw {
		var text string
		if err := json.Unmarshal(item, &text); err == nil {
			out = append(out, Option{Label: text, Value: text})
			continue
		}
		var opt Option
		if err := json.Unmarshal(item, &opt); err != nil {
			return err
		}
		if opt.Label == "" {
			opt.Label = opt.Value
		}
		out = append(out, opt)
	}
	*o = out
	return nil
}

type Field struct {
	Key         string     `json:"key"`
	Label       string     `json:"label"`
	Type        string     `json:"type"`
	Default     any        `json:"default"`
	Required    bool       `json:"required"`
	Placeholder string     `json:"placeholder"`
	Help        string     `json:"help"`
	Options     OptionList `json:"options"`
	Min         any        `json:"min"`
	Max         any        `json:"max"`
	Step        any        `json:"step"`
	Multiline   bool       `json:"multiline"`
	Important   bool       `json:"important"`
	Sort        string     `json:"sort"`
}

// ResultSchema 声明结果如何渲染：table（数组/对象表格）、links（链接）或 keyvalue（默认）。
type ResultSchema struct {
	Type    string  `json:"type"`
	Columns []Field `json:"columns"`
}

type Action struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Command     []string      `json:"command"` // CLI 桥用：该动作要执行的命令（可含 {占位符}）
	Params      []Field       `json:"params"`  // v2：动作级参数；为空时回退到顶层 params
	Result      *ResultSchema `json:"result"`
}

// Discovery 声明该平台“任务列表/发现”的展示 schema（列表接口返回原始字段，前端按此渲染）。
type Discovery struct {
	Label    string            `json:"label"`
	Columns  []Field           `json:"columns"`
	Actions  []string          `json:"actions"`
	ParamMap map[string]string `json:"param_map"` // 执行参数 key -> 条目字段 key
	Primary  string            `json:"primary"`   // 卡片视图主字段（列 key），缺省用第一列
}

// Manifest 对应脚本目录下的 omnitask.json。
type Manifest struct {
	Protocol      int        `json:"protocol"` // 0/1 视为旧协议；当前为 2
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Version       string     `json:"version"`
	Author        string     `json:"author"`
	Icon          string     `json:"icon"` // 可选 MDI 图标名，前端缺省兜底
	Runtime       string     `json:"runtime"`
	Entry         string     `json:"entry"`
	Module        string     `json:"module"` // 与 entry 二选一：用 python -m module 启动
	Args          []string   `json:"args"`
	Requires      []string   `json:"requires"`
	Permissions   []string   `json:"permissions"`    // 透明声明：network/filesystem/...（不强制执行）
	ProgressRegex string     `json:"progress_regex"` // CLI 桥从输出提取进度的正则（可选）
	Actions       []Action   `json:"actions"`
	Credentials   []Field    `json:"credentials"`
	Params        []Field    `json:"params"`
	Discovery     *Discovery `json:"discovery"`
}

type Script struct {
	Manifest     Manifest
	Dir          string
	ManifestJSON string
	ManifestHash string
	VersionKey   string // 版本化目录 key；legacy 布局为空
}

type activeVersion struct {
	Active   string `json:"active"`
	Previous string `json:"previous"`
}

// ValidateManifest 做安装/加载时的基础校验；返回第一条可读错误。
func ValidateManifest(m *Manifest) error {
	if m.Protocol > ProtocolVersion {
		return fmt.Errorf("%s 协议版本 %d 高于当前支持版本 %d", ManifestFileName, m.Protocol, ProtocolVersion)
	}
	if m.ID == "" {
		return fmt.Errorf("%s 缺少 id", ManifestFileName)
	}
	if m.Runtime == "" {
		return fmt.Errorf("%s: %s 缺少 runtime", m.ID, ManifestFileName)
	}
	if m.Entry == "" && m.Module == "" {
		return fmt.Errorf("%s: %s 需要 entry 或 module", m.ID, ManifestFileName)
	}
	p := m.Protocol
	if p == 0 {
		p = 1
	}
	if p >= 2 && len(m.Actions) == 0 {
		return fmt.Errorf("%s: 协议 v2 至少需要一个 action", m.ID)
	}
	seen := map[string]bool{}
	for _, a := range m.Actions {
		if a.ID == "" {
			return fmt.Errorf("%s: action 缺少 id", m.ID)
		}
		if seen[a.ID] {
			return fmt.Errorf("%s: action id 重复: %s", m.ID, a.ID)
		}
		seen[a.ID] = true
		if err := validateFields(a.Params, "action "+a.ID+" 参数"); err != nil {
			return err
		}
	}
	if err := validateFields(m.Params, "参数"); err != nil {
		return err
	}
	if err := validateFields(m.Credentials, "凭据"); err != nil {
		return err
	}
	if m.Discovery != nil {
		if err := validateFields(m.Discovery.Columns, "发现列"); err != nil {
			return err
		}
		for _, id := range m.Discovery.Actions {
			if !seen[id] {
				return fmt.Errorf("%s: discovery.actions 引用了不存在的 action: %s", m.ID, id)
			}
		}
	}
	return nil
}

func validateFields(fields []Field, scope string) error {
	seen := map[string]bool{}
	for _, f := range fields {
		if f.Key == "" {
			return fmt.Errorf("%s 存在空 key", scope)
		}
		if seen[f.Key] {
			return fmt.Errorf("%s 存在重复 key: %s", scope, f.Key)
		}
		seen[f.Key] = true
	}
	return nil
}

// LoadScripts 扫描 scripts_dir 下每个子目录的 omnitask.json。
// 单个脚本损坏时跳过，不影响其他脚本加载。
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
		root := filepath.Join(dir, entry.Name())
		versionDir, versionKey, ok := activeVersionDir(root)
		if !ok {
			versionDir, versionKey = root, ""
		}
		path := filepath.Join(versionDir, ManifestFileName)
		buf, err := os.ReadFile(path)
		if err != nil {
			continue // 非脚本目录
		}
		raw := bytes.TrimPrefix(buf, []byte{0xEF, 0xBB, 0xBF})
		var m Manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if ValidateManifest(&m) != nil {
			continue
		}
		sum := sha256.Sum256(raw)
		out = append(out, Script{
			Manifest:     m,
			Dir:          versionDir,
			ManifestJSON: string(raw),
			ManifestHash: hex.EncodeToString(sum[:]),
			VersionKey:   versionKey,
		})
	}
	return out, nil
}

// activeVersionDir 优先读取版本化布局；active.json 存在但无效时返回 false，
// 由调用方回退到 legacy 的 <id>/omnitask.json 布局。
func activeVersionDir(root string) (string, string, bool) {
	buf, err := os.ReadFile(filepath.Join(root, "active.json"))
	if err != nil {
		return "", "", false
	}
	var active activeVersion
	if err := json.Unmarshal(buf, &active); err != nil || active.Active == "" {
		return "", "", false
	}
	dir := filepath.Join(root, "versions", active.Active)
	if _, err := os.Stat(filepath.Join(dir, ManifestFileName)); err != nil {
		return "", "", false
	}
	return dir, active.Active, true
}
