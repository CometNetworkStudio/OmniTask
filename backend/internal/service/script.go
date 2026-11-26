package service

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	agentv1 "backend/api/agent/v1"
	v1 "backend/api/v1"
	"backend/internal/agenthost"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
)

// InstallJob 记录一次脚本安装的进度。
type InstallJob struct {
	ID      string
	Status  string // running | succeeded | failed
	Stage   string
	Percent int
	Message string
	Error   string
	Script  *v1.ScriptResponse
}

// ScriptService 负责平台脚本的安装（GitHub zip）、注册表同步与启停。
type ScriptService struct {
	repo         *repository.ScriptRepository
	jobs         *repository.ScriptInstallJobRepository
	agent        *AgentClient
	scriptsDir   string
	python       string
	autoInstall  bool
	keepVersions int
	pipIndex     string
	sid          *sid.Sid
	logger       *log.Logger

	syncMu   sync.Mutex
	lastSync time.Time
}

func NewScriptService(
	repo *repository.ScriptRepository,
	jobs *repository.ScriptInstallJobRepository,
	agent *AgentClient,
	conf *viper.Viper,
	sid *sid.Sid,
	logger *log.Logger,
) *ScriptService {
	_ = jobs.MarkRunningInterrupted(context.Background())
	return &ScriptService{
		repo:         repo,
		jobs:         jobs,
		agent:        agent,
		scriptsDir:   conf.GetString("agent.scripts_dir"),
		python:       conf.GetString("agent.python"),
		autoInstall:  conf.GetBool("agent.auto_install_deps"),
		keepVersions: conf.GetInt("agent.keep_versions"),
		pipIndex:     conf.GetString("agent.pip_index_url"),
		sid:          sid,
		logger:       logger,
	}
}

// Sync 以 Host 实际加载的 manifest 为准刷新注册表，解决安装快照漂移。
// Host 不可达时保持现状，不误标记脚本缺失。
func (s *ScriptService) Sync(ctx context.Context) {
	if s.agent == nil {
		return
	}
	s.syncMu.Lock()
	if time.Since(s.lastSync) < 3*time.Second {
		s.syncMu.Unlock()
		return
	}
	s.lastSync = time.Now()
	s.syncMu.Unlock()
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := s.agent.ListScripts(cctx)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, item := range resp.GetItems() {
		id := item.GetId()
		if id == "" {
			continue
		}
		seen[id] = true
		status := "installed"
		lastError := ""
		if !item.GetAvailable() {
			status = "missing_runtime"
			lastError = item.GetError()
		}
		script := &model.Script{
			ID: id, Name: item.GetName(), Version: item.GetVersion(),
			Manifest: item.GetManifest(), ManifestHash: item.GetHash(),
			Enabled: true, Status: status, LastError: lastError,
			InstalledAt: time.Now(), UpdatedAt: time.Now(),
		}
		if existing, err := s.repo.Get(ctx, id); err == nil {
			if existing.ManifestHash == item.GetHash() && existing.Status == status && existing.LastError == lastError {
				continue
			}
			script.SourceURL = existing.SourceURL
			script.Ref = existing.Ref
			script.CommitSHA = existing.CommitSHA
			script.Enabled = existing.Enabled
			script.InstalledAt = existing.InstalledAt
		}
		if err := s.repo.Upsert(ctx, script); err != nil {
			s.logger.Warn("sync script failed: " + id)
		}
	}
	scripts, err := s.repo.List(ctx)
	if err != nil {
		return
	}
	for i := range scripts {
		if !seen[scripts[i].ID] && scripts[i].Status != "missing" {
			_ = s.repo.UpdateStatus(ctx, scripts[i].ID, "missing", "Host 未加载该脚本")
		}
	}
}

func (s *ScriptService) List(ctx context.Context) ([]v1.ScriptResponse, error) {
	s.Sync(ctx)
	scripts, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]v1.ScriptResponse, 0, len(scripts))
	for i := range scripts {
		resp := toScriptResponse(&scripts[i])
		resp.HasBackup = s.hasBackup(resp.ID)
		out = append(out, *resp)
	}
	return out, nil
}

func (s *ScriptService) Get(ctx context.Context, id string) (*v1.ScriptResponse, error) {
	s.Sync(ctx)
	script, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, v1.ErrNotFound
	}
	resp := toScriptResponse(script)
	resp.HasBackup = s.hasBackup(id)
	return resp, nil
}

type activeScriptVersion struct {
	Active    string    `json:"active"`
	Previous  string    `json:"previous"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *ScriptService) scriptRoot(id string) string {
	return filepath.Join(s.scriptsDir, id)
}

func (s *ScriptService) activePath(id string) string {
	return filepath.Join(s.scriptRoot(id), "active.json")
}

func (s *ScriptService) versionDir(id, key string) string {
	return filepath.Join(s.scriptRoot(id), "versions", key)
}

func (s *ScriptService) readActive(id string) (activeScriptVersion, error) {
	var active activeScriptVersion
	buf, err := os.ReadFile(s.activePath(id))
	if err != nil {
		return active, err
	}
	err = json.Unmarshal(buf, &active)
	return active, err
}

func (s *ScriptService) writeActive(id string, active activeScriptVersion) error {
	active.UpdatedAt = time.Now()
	buf, err := json.MarshalIndent(active, "", "  ")
	if err != nil {
		return err
	}
	path := s.activePath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// pruneVersions 保留 active/previous 和最近若干个版本，避免无限占用磁盘。
func (s *ScriptService) pruneVersions(id, active, previous string) {
	limit := s.keepVersions
	if limit <= 0 {
		limit = 3
	}
	dir := filepath.Join(s.scriptRoot(id), "versions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type versionEntry struct {
		name string
		mod  time.Time
	}
	items := make([]versionEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, versionEntry{name: entry.Name(), mod: info.ModTime()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	keep := map[string]bool{active: true, previous: true}
	kept := 0
	if active != "" {
		kept++
	}
	if previous != "" && previous != active {
		kept++
	}
	for _, item := range items {
		if keep[item.name] {
			continue
		}
		// 只清理超过 24 小时的旧版本，避免删除仍在运行任务使用的目录。
		if kept >= limit && time.Since(item.mod) > 24*time.Hour {
			_ = os.RemoveAll(filepath.Join(dir, item.name))
			continue
		}
		keep[item.name] = true
		kept++
	}
}

func (s *ScriptService) hasBackup(id string) bool {
	active, err := s.readActive(id)
	if err != nil || active.Previous == "" {
		return false
	}
	_, err = os.Stat(filepath.Join(s.versionDir(id, active.Previous), agenthost.ManifestFileName))
	return err == nil
}

// Rollback 在版本化目录中切换 active/previous，不删除任何版本。
func (s *ScriptService) Rollback(ctx context.Context, id string) (*v1.ScriptResponse, error) {
	active, err := s.readActive(id)
	if err != nil || active.Previous == "" {
		return nil, fmt.Errorf("%w: 没有可回滚的版本", v1.ErrBadRequest)
	}
	if _, err := os.Stat(filepath.Join(s.versionDir(id, active.Previous), agenthost.ManifestFileName)); err != nil {
		return nil, fmt.Errorf("%w: 上一版本文件不存在", v1.ErrBadRequest)
	}
	active.Active, active.Previous = active.Previous, active.Active
	if err := s.writeActive(id, active); err != nil {
		return nil, err
	}
	s.syncMu.Lock()
	s.lastSync = time.Time{}
	s.syncMu.Unlock()
	return s.Get(ctx, id)
}

// InstallAsync 启动一次异步安装，立即返回 jobID；进度落库，可用 Job 查询。
func (s *ScriptService) InstallAsync(rawURL, ref string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", v1.ErrBadRequest
	}
	id, err := s.sid.GenString()
	if err != nil {
		return "", err
	}
	job := &model.ScriptInstallJob{
		ID: id, Status: "running", Stage: "queued", Percent: 0,
		Message: "排队中", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := s.jobs.Upsert(context.Background(), job); err != nil {
		return "", err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		var lastReport time.Time
		report := func(percent int, stage, message string) {
			now := time.Now()
			if now.Sub(lastReport) < 200*time.Millisecond {
				return
			}
			lastReport = now
			job.Percent = percent
			job.Stage = stage
			job.Message = message
			job.UpdatedAt = now
			_ = s.jobs.Upsert(context.Background(), job)
		}
		resp, err := s.install(ctx, rawURL, ref, report)
		job.UpdatedAt = time.Now()
		if err != nil {
			job.Status = "failed"
			job.Percent = 100
			job.Message = "安装失败"
			job.Error = err.Error()
			_ = s.jobs.Upsert(context.Background(), job)
			return
		}
		job.Status = "succeeded"
		job.Percent = 100
		job.Stage = "done"
		job.Message = "安装完成"
		job.ScriptID = resp.ID
		_ = s.jobs.Upsert(context.Background(), job)
		s.syncMu.Lock()
		s.lastSync = time.Time{}
		s.syncMu.Unlock()
	}()
	return id, nil
}

// Job 返回安装任务状态。
func (s *ScriptService) Job(id string) (*InstallJob, bool) {
	modelJob, err := s.jobs.Get(context.Background(), id)
	if err != nil {
		return nil, false
	}
	job := &InstallJob{
		ID: modelJob.ID, Status: modelJob.Status, Stage: modelJob.Stage,
		Percent: modelJob.Percent, Message: modelJob.Message, Error: modelJob.Error,
	}
	if modelJob.ScriptID != "" {
		if script, err := s.repo.Get(context.Background(), modelJob.ScriptID); err == nil {
			job.Script = toScriptResponse(script)
		}
	}
	return job, true
}

// Reinstall 使用已记录的来源与 ref 重新安装，用于更新脚本。
func (s *ScriptService) Reinstall(ctx context.Context, id string) (string, error) {
	script, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", v1.ErrNotFound
	}
	if script.SourceURL == "" {
		return "", fmt.Errorf("%w: 该脚本没有来源地址，无法重新安装", v1.ErrBadRequest)
	}
	return s.InstallAsync(script.SourceURL, script.Ref)
}

// Debug 在平台上直接运行一次脚本，供管理员排查问题。
func (s *ScriptService) Debug(ctx context.Context, id string, req *v1.ScriptDebugRequest) ([]v1.ScriptDebugEvent, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, v1.ErrNotFound
	}
	if s.agent == nil {
		return nil, fmt.Errorf("%w: Agent 未连接", v1.ErrBadRequest)
	}
	taskID, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	events := make([]v1.ScriptDebugEvent, 0, 32)
	err = s.agent.Execute(cctx, &agentv1.ExecuteRequest{
		TaskId: taskID, Platform: id, Action: req.Action,
		Params: req.Params, Credentials: req.Credentials, Proxy: req.Proxy,
	}, func(ev *agentv1.ExecuteEvent) {
		events = append(events, debugEvent(ev))
	})
	if err != nil {
		return nil, err
	}
	return events, nil
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
	rawManifest := bytes.TrimPrefix(buf, []byte{0xEF, 0xBB, 0xBF})
	var manifest agenthost.Manifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", agenthost.ManifestFileName, err)
	}
	if err := agenthost.ValidateManifest(&manifest); err != nil {
		return nil, err
	}
	if manifest.Entry != "" {
		if _, err := os.Stat(filepath.Join(filepath.Dir(manifestPath), manifest.Entry)); err != nil {
			return nil, fmt.Errorf("entry 文件不存在: %s", manifest.Entry)
		}
	}

	report(68, "copy", "写入脚本目录")
	if err := os.MkdirAll(s.scriptsDir, 0o755); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(rawManifest)
	manifestHash := hex.EncodeToString(sum[:])
	versionKey := fmt.Sprintf("%s-%d", manifestHash[:12], time.Now().UnixNano())
	versionDir := s.versionDir(manifest.ID, versionKey)
	if err := os.MkdirAll(filepath.Dir(versionDir), 0o755); err != nil {
		return nil, err
	}
	if err := copyDir(filepath.Dir(manifestPath), versionDir); err != nil {
		_ = os.RemoveAll(versionDir)
		return nil, err
	}

	if s.autoInstall {
		if err := s.installDeps(ctx, versionDir, func(pct int, msg string) {
			report(74+pct*23/100, "deps", msg)
		}); err != nil {
			_ = os.RemoveAll(versionDir)
			return nil, err
		}
	}

	active, _ := s.readActive(manifest.ID)
	previous := active.Active
	if previous == versionKey {
		previous = active.Previous
	}
	if err := s.writeActive(manifest.ID, activeScriptVersion{
		Active: versionKey, Previous: previous,
	}); err != nil {
		return nil, err
	}
	s.pruneVersions(manifest.ID, versionKey, previous)

	report(98, "register", "写入注册表")
	status := "installed"
	if _, err := exec.LookPath(s.runtimeFor(manifest)); err != nil {
		status = "missing_runtime"
	}
	now := time.Now()
	storedRef := ref
	if storedRef == "" {
		storedRef = "main"
	}
	commit := ""
	if isHexRef(ref) {
		commit = ref
	}
	script := &model.Script{
		ID: manifest.ID, Name: manifest.Name, Version: manifest.Version,
		SourceURL: rawURL, Ref: storedRef, CommitSHA: commit,
		ManifestHash: manifestHash, Manifest: string(rawManifest),
		Enabled: true, Status: status, InstalledAt: now, UpdatedAt: now,
	}
	if existing, err := s.repo.Get(ctx, manifest.ID); err == nil {
		script.InstalledAt = existing.InstalledAt
		script.Enabled = existing.Enabled
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
		refPath := "main"
		if ref != "" {
			refPath = ref
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
	_ = os.RemoveAll(s.scriptRoot(id))
	_ = os.RemoveAll(filepath.Join(filepath.Dir(s.scriptsDir), ".script_backups", id))
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
		SourceURL: script.SourceURL, Ref: script.Ref, CommitSHA: script.CommitSHA,
		ManifestHash: script.ManifestHash, Manifest: script.Manifest,
		LastError: script.LastError, Enabled: script.Enabled, Status: script.Status,
		InstalledAt: script.InstalledAt, UpdatedAt: script.UpdatedAt,
	}
	var m agenthost.Manifest
	if script.Manifest != "" && json.Unmarshal([]byte(script.Manifest), &m) == nil {
		resp.Runtime = m.Runtime
		resp.Author = m.Author
		resp.Icon = m.Icon
		for _, a := range m.Actions {
			resp.Actions = append(resp.Actions, a.Name)
		}
	}
	return resp
}

func debugEvent(ev *agentv1.ExecuteEvent) v1.ScriptDebugEvent {
	kind := "progress"
	switch ev.GetType() {
	case agentv1.EventType_EVENT_TYPE_STARTED:
		kind = "started"
	case agentv1.EventType_EVENT_TYPE_LOG:
		kind = "log"
	case agentv1.EventType_EVENT_TYPE_STEP:
		kind = "step"
	case agentv1.EventType_EVENT_TYPE_METRIC:
		kind = "metric"
	case agentv1.EventType_EVENT_TYPE_SUCCEEDED:
		kind = "succeeded"
	case agentv1.EventType_EVENT_TYPE_FAILED:
		kind = "failed"
	}
	var result any
	if ev.GetResultJson() != "" {
		_ = json.Unmarshal([]byte(ev.GetResultJson()), &result)
	} else if len(ev.GetResult()) > 0 {
		result = ev.GetResult()
	}
	return v1.ScriptDebugEvent{
		Kind: kind, Percent: int(ev.GetPercent()), Message: ev.GetMessage(),
		Level: ev.GetLevel(), Step: ev.GetStep(), Label: ev.GetLabel(),
		StepStatus: ev.GetStatus(), Current: int(ev.GetCurrent()), Total: int(ev.GetTotal()),
		Unit: ev.GetUnit(), Key: ev.GetKey(), Value: ev.GetValue(), Stage: ev.GetStage(),
		Timestamp: ev.GetTimestamp(), Result: result,
	}
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
		if !info.IsDir() && info.Name() == agenthost.ManifestFileName {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("压缩包中未找到 %s", agenthost.ManifestFileName)
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
