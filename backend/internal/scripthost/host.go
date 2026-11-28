package scripthost

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	hostv1 "backend/api/host/v1"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Version = "host-go/0.1.0"

// Host 是 Go 脚本宿主：扫描脚本目录，按 manifest 以子进程 + stdio JSON 调用脚本，
// 并实现 gRPC HostService 供后端调用。支持热扫描（安装后无需重启）。
type Host struct {
	hostv1.UnimplementedHostServiceServer
	mu            sync.RWMutex
	scripts       map[string]Script
	available     map[string]bool
	runtimes      map[string]string
	runtimeErrors map[string]string

	dir      string
	python   string
	sdkDir   string
	bankAddr string
	logger   *log.Logger

	workspacesDir   string
	rateDir         string
	processSem      chan struct{}
	platformLimit   int
	platformSems    map[string]chan struct{}
	workspaceMu     sync.Mutex
	keepWorkspaces  bool
	keepFailedWorks bool
}

func NewHost(conf *viper.Viper, logger *log.Logger) (*Host, error) {
	workspacesDir := absolute(conf.GetString("host.workspaces_dir"))
	if workspacesDir == "" {
		workspacesDir = absolute("workspaces")
	}
	rateDir := absolute(conf.GetString("host.rate_dir"))
	if rateDir == "" {
		rateDir = absolute("rate")
	}
	maxProcesses := conf.GetInt("host.max_processes")
	if maxProcesses <= 0 {
		maxProcesses = 8
	}
	platformLimit := conf.GetInt("host.platform_concurrency")
	if platformLimit <= 0 {
		platformLimit = 2
	}
	h := &Host{
		scripts:         map[string]Script{},
		available:       map[string]bool{},
		runtimes:        map[string]string{},
		runtimeErrors:   map[string]string{},
		dir:             absolute(conf.GetString("host.scripts_dir")),
		python:          conf.GetString("host.python"),
		sdkDir:          absolute(conf.GetString("host.pythonpath")),
		bankAddr:        conf.GetString("host.bank_addr"),
		logger:          logger,
		workspacesDir:   workspacesDir,
		rateDir:         rateDir,
		processSem:      make(chan struct{}, maxProcesses),
		platformLimit:   platformLimit,
		platformSems:    map[string]chan struct{}{},
		keepWorkspaces:  conf.GetBool("host.keep_workspaces"),
		keepFailedWorks: conf.GetBool("host.keep_failed_workspaces"),
	}
	if err := os.MkdirAll(h.dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(workspacesDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(rateDir, 0o755); err != nil {
		return nil, err
	}
	h.reload()
	return h, nil
}

func (h *Host) platformSem(platform string) chan struct{} {
	h.workspaceMu.Lock()
	defer h.workspaceMu.Unlock()
	sem, ok := h.platformSems[platform]
	if !ok {
		sem = make(chan struct{}, h.platformLimit)
		h.platformSems[platform] = sem
	}
	return sem
}

func (h *Host) acquire(ctx context.Context, platform string) (func(), bool) {
	select {
	case h.processSem <- struct{}{}:
	case <-ctx.Done():
		return nil, false
	}
	platformSem := h.platformSem(platform)
	select {
	case platformSem <- struct{}{}:
	case <-ctx.Done():
		<-h.processSem
		return nil, false
	}
	return func() {
		<-platformSem
		<-h.processSem
	}, true
}

func (h *Host) reload() {
	loaded, err := LoadScripts(h.dir)
	if err != nil {
		h.logger.Warn("scan scripts dir failed", zap.String("dir", h.dir), zap.Error(err))
		return
	}
	scripts := map[string]Script{}
	available := map[string]bool{}
	runtimes := map[string]string{}
	runtimeErrors := map[string]string{}
	for _, s := range loaded {
		scripts[s.Manifest.ID] = s
		runtimePath, err := h.resolveRuntime(s.Manifest, s.Dir)
		if err != nil {
			h.logger.Warn("script runtime not found",
				zap.String("script", s.Manifest.ID), zap.String("runtime", s.Manifest.Runtime))
			available[s.Manifest.ID] = false
			runtimeErrors[s.Manifest.ID] = err.Error()
			continue
		}
		runtimes[s.Manifest.ID] = runtimePath
		available[s.Manifest.ID] = true
	}
	h.mu.Lock()
	h.scripts, h.available, h.runtimes, h.runtimeErrors = scripts, available, runtimes, runtimeErrors
	h.mu.Unlock()
}

func (h *Host) resolveRuntime(m Manifest, dir string) (string, error) {
	name := m.Runtime
	if strings.HasPrefix(name, "python") {
		// 优先使用脚本自带 venv（依赖自动安装生成）
		for _, candidate := range []string{
			filepath.Join(dir, ".venv", "bin", "python"),
			filepath.Join(dir, ".venv", "Scripts", "python.exe"),
		} {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
		if h.python != "" {
			name = h.python
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Abs(path)
}

func (h *Host) lookup(id string) (Script, string, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	script, ok := h.scripts[id]
	if !ok || !h.available[id] {
		return Script{}, "", false
	}
	return script, h.runtimes[id], true
}

func (h *Host) Health(ctx context.Context, req *hostv1.HealthRequest) (*hostv1.HealthResponse, error) {
	h.reload()
	h.mu.RLock()
	defer h.mu.RUnlock()
	platforms := make([]string, 0, len(h.available))
	for id, ok := range h.available {
		if ok {
			platforms = append(platforms, id)
		}
	}
	sort.Strings(platforms)
	return &hostv1.HealthResponse{Version: Version, Platforms: platforms}, nil
}

// ListScripts 返回 Host 实际加载的脚本清单，后端以此同步注册表。
func (h *Host) ListScripts(ctx context.Context, req *hostv1.ListScriptsRequest) (*hostv1.ListScriptsResponse, error) {
	h.reload()
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.scripts))
	for id := range h.scripts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	resp := &hostv1.ListScriptsResponse{}
	for _, id := range ids {
		s := h.scripts[id]
		item := &hostv1.ScriptInfo{
			Id:          id,
			Name:        s.Manifest.Name,
			Version:     s.Manifest.Version,
			Author:      s.Manifest.Author,
			Icon:        s.Manifest.Icon,
			Runtime:     s.Manifest.Runtime,
			Manifest:    s.ManifestJSON,
			Hash:        s.ManifestHash,
			Available:   h.available[id],
			Error:       h.runtimeErrors[id],
			RuntimePath: h.runtimes[id],
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

type scriptRequest struct {
	Type        string            `json:"type"`
	TaskID      string            `json:"task_id"`
	Platform    string            `json:"platform"`
	Action      string            `json:"action"`
	Params      map[string]string `json:"params"`
	Credentials map[string]string `json:"credentials"`
	BankAddr    string            `json:"bank_addr"`
	Proxy       string            `json:"proxy"`
}

type scriptEvent struct {
	Type    string `json:"type"`
	TaskID  string `json:"task_id"`
	Event   string `json:"event"`
	Percent *int   `json:"percent"`
	Message string `json:"message"`
	Result  any    `json:"result"`
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Unit    string `json:"unit"`
	Stage   string `json:"stage"`
	Step    string `json:"step"`
	Label   string `json:"label"`
	Status  string `json:"status"`
	Level   string `json:"level"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func (h *Host) Execute(req *hostv1.ExecuteRequest, stream hostv1.HostService_ExecuteServer) error {
	h.reload()
	taskID := req.GetTaskId()
	script, runtimePath, ok := h.lookup(req.GetPlatform())
	if !ok {
		return h.fail(stream, taskID, "platform unavailable: "+req.GetPlatform())
	}
	release, ok := h.acquire(stream.Context(), req.GetPlatform())
	if !ok {
		return h.fail(stream, taskID, "任务已取消")
	}
	defer release()

	workspace, err := h.prepareWorkspace("task-"+taskID, script)
	if err != nil {
		return h.fail(stream, taskID, err.Error())
	}
	succeeded := false
	defer func() { h.cleanupWorkspace(workspace, succeeded) }()

	cmd := exec.Command(runtimePath, commandArgs(script.Manifest)...)
	cmd.Dir = workspace
	cmd.Env = h.scriptEnv(baseEnv(), script, workspace, taskID, req.GetProxy())

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return h.fail(stream, taskID, err.Error())
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return h.fail(stream, taskID, err.Error())
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return h.fail(stream, taskID, err.Error())
	}
	payload, _ := json.Marshal(scriptRequest{
		Type:        "execute",
		TaskID:      taskID,
		Platform:    req.GetPlatform(),
		Action:      req.GetAction(),
		Params:      req.GetParams(),
		Credentials: req.GetCredentials(),
		BankAddr:    h.bankAddr,
		Proxy:       req.GetProxy(),
	})
	_, _ = stdin.Write(append(payload, '\n'))

	// 取消协议：先通知脚本清理，宽限期后再强杀。
	done := make(chan struct{})
	go func() {
		select {
		case <-stream.Context().Done():
			_, _ = stdin.Write([]byte(`{"type":"cancel"}` + "\n"))
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			}
		case <-done:
		}
	}()

	terminal := false
	badLines := 0
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev scriptEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil || (ev.Type != "event" && ev.Type != "ready") {
			badLines++
			if badLines == 1 {
				_ = stream.Send(&hostv1.ExecuteEvent{
					TaskId:    taskID,
					Type:      hostv1.EventType_EVENT_TYPE_LOG,
					Level:     "warn",
					Message:   "脚本 stdout 出现非协议输出，已忽略；stdout 只能逐行输出 JSON，调试信息请走 stderr",
					Timestamp: time.Now().Unix(),
				})
			}
			continue
		}
		if ev.Type == "ready" {
			continue
		}
		percent := 0
		hasPercent := ev.Percent != nil
		if hasPercent {
			percent = *ev.Percent
		} else if ev.Total > 0 {
			percent = ev.Current * 100 / ev.Total
			hasPercent = true
		}
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		resultJSON := ""
		if ev.Result != nil {
			if buf, err := json.Marshal(ev.Result); err == nil {
				resultJSON = string(buf)
			}
		}
		event := &hostv1.ExecuteEvent{
			TaskId:     taskID,
			Percent:    int32(percent),
			HasPercent: hasPercent,
			Message:    ev.Message,
			Timestamp:  time.Now().Unix(),
			Result:     stringMap(ev.Result),
			ResultJson: resultJSON,
			Step:       ev.Step,
			Label:      ev.Label,
			Status:     ev.Status,
			Level:      ev.Level,
			Current:    int32(ev.Current),
			Total:      int32(ev.Total),
			Unit:       ev.Unit,
			Key:        ev.Key,
			Value:      ev.Value,
			Stage:      ev.Stage,
		}
		switch ev.Event {
		case "succeeded":
			event.Type = hostv1.EventType_EVENT_TYPE_SUCCEEDED
			terminal = true
			succeeded = true
		case "failed":
			event.Type = hostv1.EventType_EVENT_TYPE_FAILED
			terminal = true
		case "started":
			event.Type = hostv1.EventType_EVENT_TYPE_STARTED
		case "log":
			event.Type = hostv1.EventType_EVENT_TYPE_LOG
		case "step":
			event.Type = hostv1.EventType_EVENT_TYPE_STEP
		case "metric":
			event.Type = hostv1.EventType_EVENT_TYPE_METRIC
		case "heartbeat":
			event.Type = hostv1.EventType_EVENT_TYPE_HEARTBEAT
		default:
			event.Type = hostv1.EventType_EVENT_TYPE_PROGRESS
		}
		if err := stream.Send(event); err != nil {
			break
		}
		if terminal {
			break
		}
	}
	_ = cmd.Wait()
	close(done)
	_ = stdin.Close()
	if !terminal {
		msg := "script exited without terminal event"
		if s := lastLine(stderr.String()); s != "" {
			msg += ": " + s
		}
		return h.fail(stream, taskID, msg)
	}
	return nil
}

func commandArgs(m Manifest) []string {
	args := make([]string, 0, len(m.Args)+2)
	if m.Module != "" {
		args = append(args, "-m", m.Module)
	} else {
		args = append(args, m.Entry)
	}
	return append(args, m.Args...)
}

type discoverRequest struct {
	Type        string            `json:"type"`
	Platform    string            `json:"platform"`
	Params      map[string]string `json:"params"`
	Credentials map[string]string `json:"credentials"`
	Proxy       string            `json:"proxy"`
}

// Discover 让脚本列出该账号在平台上的条目（@listing）。
func (h *Host) Discover(ctx context.Context, req *hostv1.DiscoverRequest) (*hostv1.DiscoverResponse, error) {
	h.reload()
	script, runtimePath, ok := h.lookup(req.GetPlatform())
	if !ok {
		return nil, status.Error(codes.Unavailable, "platform unavailable: "+req.GetPlatform())
	}
	release, ok := h.acquire(ctx, req.GetPlatform())
	if !ok {
		return nil, status.Error(codes.Canceled, "discover canceled")
	}
	defer release()

	label := fmt.Sprintf("discover-%s-%d", req.GetPlatform(), time.Now().UnixNano())
	workspace, err := h.prepareWorkspace(label, script)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer h.cleanupWorkspace(workspace, true)

	cmd := exec.CommandContext(ctx, runtimePath, commandArgs(script.Manifest)...)
	cmd.Dir = workspace
	cmd.Env = h.scriptEnv(baseEnv(), script, workspace, label, req.GetProxy())

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	payload, _ := json.Marshal(discoverRequest{
		Type:        "discover",
		Platform:    req.GetPlatform(),
		Params:      req.GetParams(),
		Credentials: req.GetCredentials(),
		Proxy:       req.GetProxy(),
	})
	_, _ = stdin.Write(append(payload, '\n'))
	_ = stdin.Close()

	var message struct {
		Type  string `json:"type"`
		Items []struct {
			Key    string            `json:"key"`
			Fields map[string]string `json:"fields"`
		} `json:"items"`
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &message); err != nil || message.Type != "items" {
			continue
		}
		resp := &hostv1.DiscoverResponse{}
		for _, item := range message.Items {
			resp.Items = append(resp.Items, &hostv1.DiscoverItem{Key: item.Key, Fields: item.Fields})
		}
		_ = cmd.Wait()
		return resp, nil
	}
	_ = cmd.Wait()
	return nil, status.Error(codes.Internal, "discover 未返回条目: "+lastLine(stderr.String()))
}

func (h *Host) fail(stream hostv1.HostService_ExecuteServer, taskID, message string) error {
	h.logger.Warn("host execute failed", zap.String("task", taskID), zap.String("error", message))
	return stream.Send(&hostv1.ExecuteEvent{
		TaskId:    taskID,
		Type:      hostv1.EventType_EVENT_TYPE_FAILED,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}

func absolute(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// stringMap 把脚本 result 中的任意 JSON 值降级成字符串，兼容数字/布尔等。
func stringMap(value any) map[string]string {
	in, ok := value.(map[string]any)
	if !ok || in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, item := range in {
		if item == nil {
			out[key] = ""
			continue
		}
		out[key] = fmt.Sprint(item)
	}
	return out
}

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
