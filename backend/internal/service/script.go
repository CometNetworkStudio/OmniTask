package service

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	v1 "backend/api/v1"
	"backend/internal/agenthost"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
)

// InstallJob 记录一次脚本安装的进度（内存态，单实例）。
type InstallJob struct {
	ID      string
	Status  string // running | succeeded | failed
	Stage   string
	Percent int
	Message string
	Error   string
	Script  *v1.ScriptResponse
}

// ScriptService 负责平台脚本的安装（GitHub zip）、注册表与启停。
type ScriptService struct {
	repo        *repository.ScriptRepository
	scriptsDir  string
	python      string
	autoInstall bool
	pipIndex    string
	sid         *sid.Sid
	logger      *log.Logger

	jobsMu sync.Mutex
	jobs   map[string]*InstallJob
}

func NewScriptService(repo *repository.ScriptRepository, conf *viper.Viper, sid *sid.Sid, logger *log.Logger) *ScriptService {
	return &ScriptService{
		repo:        repo,
		scriptsDir:  conf.GetString("agent.scripts_dir"),
		python:      conf.GetString("agent.python"),
		autoInstall: conf.GetBool("agent.auto_install_deps"),
		pipIndex:    conf.GetString("agent.pip_index_url"),
		sid:         sid,
		logger:      logger,
		jobs:        map[string]*InstallJob{},
	}
}

func (s *ScriptService) List(ctx context.Context) ([]v1.ScriptResponse, error) {
	scripts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]v1.ScriptResponse, 0, len(scripts))
	for i := range scripts {
		out = append(out, *toScriptResponse(&scripts[i]))
	}
	return out, nil
}

// InstallAsync 启动一次异步安装，立即返回 jobID；进度用 Job 查询。
func (s *ScriptService) InstallAsync(rawURL, ref string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", v1.ErrBadRequest
	}
	id, err := s.sid.GenString()
	if err != nil {
		return "", err
	}
	job := &InstallJob{ID: id, Status: "running", Stage: "queued", Message: "排队中"}
	s.jobsMu.Lock()
	s.jobs[id] = job
	s.jobsMu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		report := func(percent int, stage, message string) {
			s.jobsMu.Lock()
			job.Percent = percent
			job.Stage = stage
			job.Message = message
			s.jobsMu.Unlock()
		}
		resp, err := s.install(ctx, rawURL, ref, report)
		s.jobsMu.Lock()
		defer s.jobsMu.Unlock()
		if err != nil {
			job.Status = "failed"
			job.Percent = 100
			job.Message = "安装失败"
			job.Error = err.Error()
			return
		}
		job.Status = "succeeded"
		job.Percent = 100
		job.Stage = "done"
		job.Message = "安装完成"
		job.Script = resp
	}()
	return id, nil
}

// Job 返回安装任务状态（副本）。
func (s *ScriptService) Job(id string) (*InstallJob, bool) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	copied := *job
	return &copied, true
}

// install 同步执行安装；report(percent, stage, message) 上报进度。
func (s *ScriptService) install(ctx context.Context, rawURL, ref string, report func(int, string, string)) (*v1.ScriptResponse, error) {
	zipURL := resolveZipURL(rawURL, ref)
	report(1, "download", "准备下载")

	tmp, err := os.MkdirTemp("", "omnitask-script-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	zipPath := filepath.Join(tmp, "script.zip")
	if err := downloadFile(ctx, zipURL, zipPath, func(pct int) {
		report(1+pct*58/100, "download", fmt.Sprintf("下载中 %d%%", pct))
	}); err != nil {
		return nil, err
	}
	report(60, "extract", "解压中")
	extractDir := filepath.Join(tmp, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		return nil, err
	}
	manifestPath, err := findManifest(extractDir)
	if err != nil {
		return nil, err
	}
	buf, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest agenthost.Manifest
	if err := json.Unmarshal(buf, &manifest); err != nil {
		return nil, fmt.Errorf("解析 script.json 失败: %w", err)
	}
	if manifest.ID == "" || (manifest.Entry == "" && manifest.Module == "") {
		return nil, errors.New("script.json 缺少 id 或 entry/module")
	}

	report(68, "copy", "写入脚本目录")
	if err := os.MkdirAll(s.scriptsDir, 0o755); err != nil {
		return nil, err
	}
	dest := filepath.Join(s.scriptsDir, manifest.ID)
	if err := os.RemoveAll(dest); err != nil {
		return nil, err
	}
	if err := copyDir(filepath.Dir(manifestPath), dest); err != nil {
		return nil, err
	}

	if s.autoInstall {
		if err := s.installDeps(ctx, dest, func(pct int, msg string) {
			report(74+pct*23/100, "deps", msg)
		}); err != nil {
			return nil, err
		}
	}

	report(98, "register", "写入注册表")
	status := "installed"
	if _, err := exec.LookPath(s.runtimeFor(manifest)); err != nil {
		status = "missing_runtime"
	}
	now := time.Now()
	manifestJSON, _ := json.Marshal(manifest)
	script := &model.Script{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version,
		SourceURL: rawURL, Manifest: string(manifestJSON), Enabled: true,
		Status: status, InstalledAt: now, UpdatedAt: now,
	}
	if err := s.repo.Upsert(ctx, script); err != nil {
		return nil, err
	}
	s.logger.Info("script installed: " + manifest.ID)
	return toScriptResponse(script), nil
}

// resolveZipURL：github.com 仓库链接统一转为 codeload 直链（避开 github.com 主站抖动），
// 已带加速前缀（如 https://gh-proxy.com/）时保留前缀。ref 为空用默认分支 main。
func resolveZipURL(raw, ref string) string {
	if strings.HasSuffix(strings.ToLower(strings.SplitN(raw, "?", 2)[0]), ".zip") {
		return raw
	}
	if i := strings.Index(raw, "github.com/"); i >= 0 {
		prefix := raw[:i]
		rest := strings.TrimSuffix(raw[i+len("github.com/"):], "/")
		rest = strings.TrimSuffix(rest, ".git")
		refPath := "refs/heads/main"
		if ref != "" {
			if isHexRef(ref) {
				refPath = ref
			} else {
				refPath = "refs/heads/" + ref
			}
		}
		return prefix + "codeload.github.com/" + rest + "/zip/" + refPath
	}
	return strings.TrimSuffix(raw, "/") + "/archive/refs/heads/main.zip"
}

func isHexRef(ref string) bool {
	if len(ref) < 7 {
		return false
	}
	for _, c := range ref {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func (s *ScriptService) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return v1.ErrNotFound
	}
	return s.repo.SetEnabled(ctx, id, enabled)
}

func (s *ScriptService) Uninstall(ctx context.Context, id string) error {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return v1.ErrNotFound
	}
	_ = os.RemoveAll(filepath.Join(s.scriptsDir, id))
	return s.repo.Delete(ctx, id)
}

// EnabledPlatforms 返回已启用脚本 id，供任务创建校验。
func (s *ScriptService) EnabledPlatforms(ctx context.Context) []string {
	ids, err := s.repo.EnabledIDs(ctx)
	if err != nil {
		return nil
	}
	return ids
}

// installDeps 在脚本目录内建独立 venv 并安装 requirements.txt（可选）；report(percent, message)。
func (s *ScriptService) installDeps(ctx context.Context, dir string, report func(int, string)) error {
	req := filepath.Join(dir, "requirements.txt")
	if _, err := os.Stat(req); err != nil {
		return nil
	}
	python := s.python
	if python == "" {
		python = "python3"
	}
	if _, err := exec.LookPath(python); err != nil {
		s.logger.Warn("skip dependency install: python not found")
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	report(5, "创建虚拟环境")
	venv := filepath.Join(dir, ".venv")
	if out, err := exec.CommandContext(cctx, python, "-m", "venv", venv).CombinedOutput(); err != nil {
		return fmt.Errorf("创建 venv 失败: %v: %s", err, lastLines(string(out), 3))
	}

	pip := filepath.Join(venv, "bin", "pip")
	if runtime.GOOS == "windows" {
		pip = filepath.Join(venv, "Scripts", "pip.exe")
	}
	args := []string{"install", "-r", req}
	if s.pipIndex != "" {
		args = append(args, "-i", s.pipIndex)
	}
	cmd := exec.CommandContext(cctx, pip, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	report(20, "安装依赖")
	percent := 20
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if percent < 92 {
			percent += 3
		}
		report(percent, line)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("安装依赖失败: %v", err)
	}
	report(100, "依赖安装完成")
	return nil
}

func lastLines(text string, n int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " | ")
}

func (s *ScriptService) runtimeFor(m agenthost.Manifest) string {
	if s.python != "" && strings.HasPrefix(m.Runtime, "python") {
		return s.python
	}
	return m.Runtime
}

func toScriptResponse(script *model.Script) *v1.ScriptResponse {
	resp := &v1.ScriptResponse{
		ID: script.ID, Name: script.Name, Version: script.Version,
		SourceURL: script.SourceURL, Enabled: script.Enabled, Status: script.Status,
		InstalledAt: script.InstalledAt, UpdatedAt: script.UpdatedAt,
	}
	var m agenthost.Manifest
	if script.Manifest != "" && json.Unmarshal([]byte(script.Manifest), &m) == nil {
		resp.Runtime = m.Runtime
		for _, a := range m.Actions {
			resp.Actions = append(resp.Actions, a.Name)
		}
	}
	return resp
}

func downloadFile(ctx context.Context, url, dest string, onProgress func(int)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d (%s)", resp.StatusCode, url)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	writer := &progressWriter{writer: out, total: resp.ContentLength, onProgress: onProgress}
	_, err = io.Copy(writer, resp.Body)
	return err
}

type progressWriter struct {
	writer     io.Writer
	written    int64
	total      int64
	onProgress func(int)
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.written += int64(n)
	if w.total > 0 && w.onProgress != nil {
		w.onProgress(int(w.written * 100 / w.total))
	}
	return n, err
}

func unzip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		target := filepath.Join(dest, file.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法压缩路径: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		out.Close()
		in.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func findManifest(root string) (string, error) {
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "script.json" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("压缩包中未找到 script.json")
	}
	return found, nil
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}
