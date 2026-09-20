package agenthost

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	agentv1 "backend/api/agent/v1"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const Version = "agent-go/0.1.0"

// Host 是 Go 脚本宿主：扫描脚本目录，按 manifest 以子进程 + stdio JSON 调用脚本，
// 并实现 gRPC AgentService 供后端调用。
type Host struct {
	agentv1.UnimplementedAgentServiceServer
	scripts   map[string]Script
	available map[string]bool
	runtimes  map[string]string
	python    string
	sdkDir    string
	bankAddr  string
	logger    *log.Logger
}

func NewHost(conf *viper.Viper, logger *log.Logger) (*Host, error) {
	scripts, err := LoadScripts(conf.GetString("agent.scripts_dir"))
	if err != nil {
		return nil, err
	}
	h := &Host{
		scripts:   map[string]Script{},
		available: map[string]bool{},
		runtimes:  map[string]string{},
		python:    conf.GetString("agent.python"),
		sdkDir:    absolute(conf.GetString("agent.pythonpath")),
		bankAddr:  conf.GetString("agent.bank_addr"),
		logger:    logger,
	}
	for _, s := range scripts {
		h.scripts[s.Manifest.ID] = s
		resolved, err := h.resolveRuntime(s.Manifest)
		if err != nil {
			logger.Warn("script runtime not found",
				zap.String("script", s.Manifest.ID), zap.String("runtime", s.Manifest.Runtime))
			h.available[s.Manifest.ID] = false
			continue
		}
		h.runtimes[s.Manifest.ID] = resolved
		h.available[s.Manifest.ID] = true
	}
	return h, nil
}

// resolveRuntime 返回绝对可执行路径（cmd.Dir 会改变相对路径基准）。
func (h *Host) resolveRuntime(m Manifest) (string, error) {
	name := m.Runtime
	if h.python != "" && strings.HasPrefix(name, "python") {
		name = h.python
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

func (h *Host) Health(ctx context.Context, req *agentv1.HealthRequest) (*agentv1.HealthResponse, error) {
	platforms := make([]string, 0, len(h.available))
	for id, ok := range h.available {
		if ok {
			platforms = append(platforms, id)
		}
	}
	sort.Strings(platforms)
	return &agentv1.HealthResponse{Version: Version, Platforms: platforms}, nil
}

type scriptRequest struct {
	Type        string            `json:"type"`
	TaskID      string            `json:"task_id"`
	Platform    string            `json:"platform"`
	Action      string            `json:"action"`
	Params      map[string]string `json:"params"`
	Credentials map[string]string `json:"credentials"`
	BankAddr    string            `json:"bank_addr"`
}

type scriptEvent struct {
	Type    string            `json:"type"`
	TaskID  string            `json:"task_id"`
	Event   string            `json:"event"`
	Percent int               `json:"percent"`
	Message string            `json:"message"`
	Result  map[string]string `json:"result"`
}

func (h *Host) Execute(req *agentv1.ExecuteRequest, stream agentv1.AgentService_ExecuteServer) error {
	taskID := req.GetTaskId()
	script, ok := h.scripts[req.GetPlatform()]
	if !ok {
		return h.fail(stream, taskID, "unknown platform: "+req.GetPlatform())
	}
	if !h.available[script.Manifest.ID] {
		return h.fail(stream, taskID, "runtime not available: "+script.Manifest.Runtime)
	}
	runtimePath := h.runtimes[script.Manifest.ID]
	if runtimePath == "" {
		return h.fail(stream, taskID, "runtime not resolved: "+script.Manifest.Runtime)
	}

	cmd := exec.CommandContext(stream.Context(), runtimePath, append([]string{script.Manifest.Entry}, script.Manifest.Args...)...)
	cmd.Dir = script.Dir
	env := os.Environ()
	if h.sdkDir != "" {
		env = append(env, "OMNITASK_SDK="+h.sdkDir, "PYTHONPATH="+h.sdkDir)
	}
	if h.bankAddr != "" {
		env = append(env, "OMNITASK_BANK_ADDR="+h.bankAddr, "OMNITASK_BANK_MODE=grpc")
	}
	cmd.Env = env

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
	})
	_, _ = stdin.Write(append(payload, '\n'))
	_ = stdin.Close()

	terminal := false
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev scriptEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil || ev.Type != "event" {
			continue
		}
		event := &agentv1.ExecuteEvent{
			TaskId:    taskID,
			Percent:   int32(ev.Percent),
			Message:   ev.Message,
			Timestamp: time.Now().Unix(),
			Result:    ev.Result,
		}
		switch ev.Event {
		case "succeeded":
			event.Type = agentv1.EventType_EVENT_TYPE_SUCCEEDED
			terminal = true
		case "failed":
			event.Type = agentv1.EventType_EVENT_TYPE_FAILED
			terminal = true
		default:
			event.Type = agentv1.EventType_EVENT_TYPE_PROGRESS
		}
		if err := stream.Send(event); err != nil {
			break
		}
		if terminal {
			break
		}
	}
	_ = cmd.Wait()
	if !terminal {
		msg := "script exited without terminal event"
		if s := lastLine(stderr.String()); s != "" {
			msg += ": " + s
		}
		return h.fail(stream, taskID, msg)
	}
	return nil
}

func (h *Host) fail(stream agentv1.AgentService_ExecuteServer, taskID, message string) error {
	h.logger.Warn("host execute failed", zap.String("task", taskID), zap.String("error", message))
	return stream.Send(&agentv1.ExecuteEvent{
		TaskId:    taskID,
		Type:      agentv1.EventType_EVENT_TYPE_FAILED,
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

func lastLine(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
