// Package authcookie 集中管理会话 Cookie 与 CSRF Cookie，避免 handler 和中间件各写一套。
package authcookie

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	SessionCookie = "omnitask_session"
	CSRFCookie    = "omnitask_csrf"
	CSRFHeader    = "X-CSRF-Token"
	// CtxUserKey / CtxSessionKey 是 gin.Context 中保存认证结果的键。
	CtxUserKey    = "auth_user"
	CtxSessionKey = "auth_session"
)

// Options 描述 Cookie 的公共属性。
type Options struct {
	TTL      time.Duration
	Secure   bool
	Domain   string
	SameSite http.SameSite
}

// NewToken 生成 256 位随机令牌，URL 安全，可直接放进 Cookie。
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Secure 判断当前请求是否应使用 Secure Cookie。
// security.cookie_secure 可显式设为 true/false；auto 时依据 TLS 或反向代理的 X-Forwarded-Proto。
func Secure(conf *viper.Viper, r *http.Request) bool {
	if conf != nil {
		switch value := conf.Get("security.cookie_secure").(type) {
		case bool:
			return value
		case string:
			switch strings.ToLower(strings.TrimSpace(value)) {
			case "true", "1", "yes":
				return true
			case "false", "0", "no":
				return false
			}
		}
	}
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// OptionsFrom 从配置和请求推导 Cookie 属性。
func OptionsFrom(conf *viper.Viper, r *http.Request) Options {
	ttlHours := 168
	if conf != nil {
		if v := conf.GetInt("security.session_ttl_hours"); v > 0 {
			ttlHours = v
		}
	}
	domain := ""
	sameSite := http.SameSiteLaxMode
	if conf != nil {
		domain = strings.TrimSpace(conf.GetString("security.cookie_domain"))
		switch strings.ToLower(strings.TrimSpace(conf.GetString("security.cookie_same_site"))) {
		case "strict":
			sameSite = http.SameSiteStrictMode
		case "none":
			sameSite = http.SameSiteNoneMode
		}
	}
	return Options{
		TTL:      time.Duration(ttlHours) * time.Hour,
		Secure:   Secure(conf, r),
		Domain:   domain,
		SameSite: sameSite,
	}
}

func WriteSession(w http.ResponseWriter, token string, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    token,
		Path:     "/",
		Domain:   opts.Domain,
		MaxAge:   int(opts.TTL.Seconds()),
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

func ClearSession(w http.ResponseWriter, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    "",
		Path:     "/",
		Domain:   opts.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

// WriteCSRF 写入双提交令牌。CSRF Cookie 必须能被前端 JS 读取，因此不能 HttpOnly。
func WriteCSRF(w http.ResponseWriter, token string, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookie,
		Value:    token,
		Path:     "/",
		Domain:   opts.Domain,
		MaxAge:   int(opts.TTL.Seconds()),
		HttpOnly: false,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

func ClearCSRF(w http.ResponseWriter, opts Options) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookie,
		Value:    "",
		Path:     "/",
		Domain:   opts.Domain,
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   opts.Secure,
		SameSite: opts.SameSite,
	})
}

func ReadCSRF(r *http.Request) string {
	cookie, err := r.Cookie(CSRFCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}
