package scripthost

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var workspaceSkip = map[string]bool{
	".venv":         true,
	".git":          true,
	"node_modules":  true,
	"__pycache__":   true,
	".pytest_cache": true,
	".mypy_cache":   true,
	".ruff_cache":   true,
	".tox":          true,
}

// prepareWorkspace 复制脚本源码到独立任务目录，避免并发任务互相覆盖 config.ini/cookie/cache。
func (h *Host) prepareWorkspace(label string, script Script) (string, error) {
	name := safeName(label)
	if name == "" {
		name = fmt.Sprintf("task-%d", os.Getpid())
	}
	dir := filepath.Join(h.workspacesDir, name)
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := copyWorkspace(script.Dir, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("准备任务工作区失败: %w", err)
	}
	for _, sub := range []string{"tmp", "home", ".cache", ".config"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			_ = os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

func (h *Host) cleanupWorkspace(dir string, succeeded bool) {
	if h.keepWorkspaces {
		return
	}
	if !succeeded && h.keepFailedWorks {
		return
	}
	_ = os.RemoveAll(dir)
}

func copyWorkspace(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if entry.IsDir() {
			if workspaceSkip[entry.Name()] {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".pyc") || strings.HasSuffix(name, ".pyo") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return copyFile(path, filepath.Join(dst, rel), info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := out.ReadFrom(in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func safeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := env[:0]
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return strings.TrimPrefix(env[i], prefix)
		}
	}
	return ""
}

// baseEnv 只传递脚本运行所需的系统变量，避免把服务端密钥、数据库连接等泄漏给脚本。
func baseEnv() []string {
	keys := []string{
		"PATH", "Path", "PATHEXT", "SystemRoot", "SYSTEMROOT", "WINDIR", "windir",
		"SystemDrive", "ComSpec", "COMSPEC", "NUMBER_OF_PROCESSORS",
		"PROCESSOR_ARCHITECTURE", "OS", "LANG", "LC_ALL", "LC_CTYPE", "TZ",
		"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy",
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		upper := strings.ToUpper(key)
		if seen[upper] {
			continue
		}
		seen[upper] = true
		if value, ok := os.LookupEnv(key); ok {
			out = append(out, key+"="+value)
		}
	}
	return out
}

// scriptEnv 注入任务工作区、SDK、题库和代理环境。
func (h *Host) scriptEnv(env []string, script Script, workspace, taskID, proxy string) []string {
	env = setEnv(env, "OMNITASK_TASK_ID", taskID)
	env = setEnv(env, "OMNITASK_SCRIPT_DIR", script.Dir)
	env = setEnv(env, "OMNITASK_WORKDIR", workspace)
	if h.sdkDir != "" {
		env = setEnv(env, "OMNITASK_SDK", h.sdkDir)
		path := h.sdkDir + string(os.PathListSeparator) + script.Dir
		if existing := envValue(env, "PYTHONPATH"); existing != "" {
			path += string(os.PathListSeparator) + existing
		}
		env = setEnv(env, "PYTHONPATH", path)
	}
	// Windows 下脚本 stdout/stdin 走管道时默认本地编码（如 cp936），
	// 与 Host 的 UTF-8 JSON 不一致，会产出 U+FFFD 乱码。
	env = setEnv(env, "PYTHONUTF8", "1")
	env = setEnv(env, "PYTHONIOENCODING", "utf-8")
	if h.bankAddr != "" {
		env = setEnv(env, "OMNITASK_BANK_ADDR", h.bankAddr)
		env = setEnv(env, "OMNITASK_BANK_MODE", "grpc")
	}
	if proxy != "" {
		env = setEnv(env, "OMNITASK_PROXY", proxy)
	}
	if h.rateDir != "" {
		env = setEnv(env, "OMNITASK_RATE_DB", filepath.Join(h.rateDir, "rate.db"))
	}
	tmp := filepath.Join(workspace, "tmp")
	home := filepath.Join(workspace, "home")
	env = setEnv(env, "TMPDIR", tmp)
	env = setEnv(env, "TEMP", tmp)
	env = setEnv(env, "TMP", tmp)
	env = setEnv(env, "HOME", home)
	env = setEnv(env, "USERPROFILE", home)
	env = setEnv(env, "APPDATA", filepath.Join(workspace, ".config"))
	env = setEnv(env, "LOCALAPPDATA", filepath.Join(workspace, ".cache"))
	env = setEnv(env, "XDG_CACHE_HOME", filepath.Join(workspace, ".cache"))
	env = setEnv(env, "XDG_CONFIG_HOME", filepath.Join(workspace, ".config"))
	return env
}
