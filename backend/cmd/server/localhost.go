package main

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"backend/pkg/log"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// startLocalHost 在 lite/单机模式下自动拉起同目录的 omnitask-host。
// 端口已被占用（已有 Host）时跳过；返回停止函数。
func startLocalHost(conf *viper.Viper, confPath string, logger *log.Logger) func() {
	noop := func() {}
	if !conf.GetBool("host.autostart") {
		return noop
	}

	bin := conf.GetString("host.bin")
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			logger.Warn("resolve server executable failed", zap.Error(err))
			return noop
		}
		dir := filepath.Dir(exe)
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		candidates := []string{
			filepath.Join(dir, "omnitask-host"+suffix),
			filepath.Join(dir, "omnitask-host-"+runtime.GOOS+"-"+runtime.GOARCH+suffix),
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				bin = candidate
				break
			}
		}
	}
	if bin == "" {
		logger.Warn("local Host binary not found, start it manually")
		return noop
	}
	if info, err := os.Stat(bin); err != nil || info.IsDir() {
		logger.Warn("local Host binary not found, start it manually", zap.String("path", bin))
		return noop
	}

	host := conf.GetString("host.listen")
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	addr := net.JoinHostPort(host, strconv.Itoa(conf.GetInt("host.port")))
	if conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond); err == nil {
		_ = conn.Close()
		logger.Info("local Host already running", zap.String("addr", addr))
		return noop
	}

	absConf, err := filepath.Abs(confPath)
	if err != nil {
		absConf = confPath
	}
	cmd := exec.Command(bin, "-conf", absConf)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env := os.Environ()
	env = append(env, "OMNITASK_CONFIG_NO_WRITE=1")
	if logFile := conf.GetString("log.log_file_name"); logFile != "" {
		env = append(env, "OMNITASK_LOG_FILE="+logFile+".host")
	}
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		logger.Warn("start local Host failed", zap.String("path", bin), zap.Error(err))
		return noop
	}
	logger.Info("local Host started", zap.String("path", bin), zap.Int("pid", cmd.Process.Pid))

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
	}
}
