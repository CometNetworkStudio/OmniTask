package middleware

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"

	v1 "backend/api/v1"
	"backend/pkg/authcookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// CSRF 对 Cookie 鉴权做两层校验：
//  1. 浏览器带来的 Origin 必须是同源或显式允许的来源；
//  2. 非安全方法必须带 X-CSRF-Token，并与双提交 Cookie 一致。
//
// 登录/注册还没有会话，只做 Origin 校验。
func CSRF(conf *viper.Viper) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, origin := range strings.Split(conf.GetString("http.cors_origins"), ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			allowed[trimmed] = true
		}
	}
	dev := conf.GetString("env") != "prod"
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" && !originAllowed(ctx, origin, allowed, dev) {
			v1.HandleError(ctx, http.StatusForbidden, v1.ErrCSRFInvalid, nil)
			ctx.Abort()
			return
		}
		switch ctx.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			ctx.Next()
			return
		}
		path := ctx.Request.URL.Path
		if strings.HasSuffix(path, "/auth/login") || strings.HasSuffix(path, "/auth/register") {
			ctx.Next()
			return
		}
		header := ctx.GetHeader(authcookie.CSRFHeader)
		cookie := authcookie.ReadCSRF(ctx.Request)
		if header == "" || cookie == "" || subtle.ConstantTimeCompare([]byte(header), []byte(cookie)) != 1 {
			v1.HandleError(ctx, http.StatusForbidden, v1.ErrCSRFInvalid, nil)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

func originAllowed(ctx *gin.Context, origin string, allowed map[string]bool, dev bool) bool {
	if allowed[origin] {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := ctx.Request.Host
	if forwarded := ctx.GetHeader("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	if strings.EqualFold(u.Host, host) {
		return true
	}
	// Vite 开发代理会保留 http://localhost:5173 作为 Origin，但后端看到的是 127.0.0.1:8080。
	// 只对来自回环地址的请求放行回环 Origin，公网请求仍然必须同源。
	if isLoopbackHostname(u.Hostname()) && isLoopbackRemote(ctx.Request.RemoteAddr) {
		return true
	}
	if dev && isLoopbackHostname(u.Hostname()) {
		return true
	}
	return false
}

func isLoopbackHostname(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return isLoopbackHostname(host)
}
