package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "backend/api/v1"
	"backend/internal/agenthost"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
)

// ScriptService 负责平台脚本的安装（GitHub zip）、注册表与启停。
type ScriptService struct {
	repo       *repository.ScriptRepository
	scriptsDir string
	python     string
	sid        *sid.Sid
	logger     *log.Logger
}

func NewScriptService(repo *repository.ScriptRepository, conf *viper.Viper, sid *sid.Sid, logger *log.Logger) *ScriptService {
	return &ScriptService{
		repo:       repo,
		scriptsDir: conf.GetString("agent.scripts_dir"),
		python:     conf.GetString("agent.python"),
		sid:        sid,
		logger:     logger,
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

// Install 下载 zip（可带加速前缀）并解压到 scripts_dir/<id>/。
func (s *ScriptService) Install(ctx context.Context, rawURL string) (*v1.ScriptResponse, error) {
	zipURL := resolveZipURL(rawURL)

	tmp, err := os.MkdirTemp("", "omnitask-script-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	zipPath := filepath.Join(tmp, "script.zip")
	if err := downloadFile(ctx, zipURL, zipPath); err != nil {
		return nil, err
	}
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
	if manifest.ID == "" || manifest.Entry == "" {
		return nil, errors.New("script.json 缺少 id 或 entry")
	}

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
// 已带加速前缀（如 https://gh-proxy.com/）时保留前缀。
func resolveZipURL(raw string) string {
	if strings.HasSuffix(strings.ToLower(strings.SplitN(raw, "?", 2)[0]), ".zip") {
		return raw
	}
	if i := strings.Index(raw, "github.com/"); i >= 0 {
		prefix := raw[:i]
		rest := strings.TrimSuffix(raw[i+len("github.com/"):], "/")
		rest = strings.TrimSuffix(rest, ".git")
		return prefix + "codeload.github.com/" + rest + "/zip/refs/heads/main"
	}
	return strings.TrimSuffix(raw, "/") + "/archive/refs/heads/main.zip"
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

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 120 * time.Second}
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
	_, err = io.Copy(out, resp.Body)
	return err
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
