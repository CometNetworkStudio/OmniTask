package server

import (
	"embed"
	"io/fs"
	stdhttp "net/http"
	"strings"

	"backend/pkg/log"
	"backend/pkg/server/http"
	"github.com/gin-gonic/gin"
)

//go:embed all:webdist
var webdistFS embed.FS

const notBuiltPage = `<!doctype html><meta charset="utf-8"><title>OmniTask</title>
<body style="font-family:sans-serif;padding:2rem"><h1>OmniTask</h1>
<p>前端尚未构建。请在 frontend 目录执行 <code>npm run build</code> 后重新编译后端，或使用开发模式（Vite）。</p></body>`

// registerStatic 托管内嵌的前端 dist（SPA），/api 前缀留给后端接口。
func registerStatic(s *http.Server, logger *log.Logger) {
	sub, err := fs.Sub(webdistFS, "webdist")
	if err != nil {
		logger.Info("static fs unavailable: " + err.Error())
		return
	}
	fileServer := stdhttp.FileServer(stdhttp.FS(sub))
	_, indexErr := fs.Stat(sub, "index.html")

	s.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.JSON(stdhttp.StatusNotFound, gin.H{"code": 404, "message": "not found", "data": gin.H{}})
			return
		}
		if path == "/" || path == "/index.html" {
			serveIndex(c, sub, indexErr)
			return
		}
		if f, err := sub.Open(strings.TrimPrefix(path, "/")); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		serveIndex(c, sub, indexErr)
	})
}

func serveIndex(c *gin.Context, sub fs.FS, indexErr error) {
	if indexErr != nil {
		c.Data(stdhttp.StatusOK, "text/html; charset=utf-8", []byte(notBuiltPage))
		return
	}
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		c.Data(stdhttp.StatusOK, "text/html; charset=utf-8", []byte(notBuiltPage))
		return
	}
	c.Data(stdhttp.StatusOK, "text/html; charset=utf-8", data)
}
