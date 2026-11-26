package authcookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestSecureAuto(t *testing.T) {
	conf := viper.New()
	conf.Set("security.cookie_secure", "auto")
	req := httptest.NewRequest("GET", "/", nil)
	if Secure(conf, req) {
		t.Fatal("plain http should not be secure in auto mode")
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	if !Secure(conf, req) {
		t.Fatal("forwarded https should be secure")
	}
	conf.Set("security.cookie_secure", "false")
	conf.Set("security.session_ttl_hours", 1)
	if Secure(conf, req) {
		t.Fatal("explicit false should win")
	}
}

func TestNewToken(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 32 {
		t.Fatalf("token too short: %d", len(token))
	}
	other, _ := NewToken()
	if token == other {
		t.Fatal("tokens should differ")
	}
}

func TestOptionsFrom(t *testing.T) {
	conf := viper.New()
	conf.Set("security.session_ttl_hours", 2)
	conf.Set("security.cookie_same_site", "strict")
	opts := OptionsFrom(conf, httptest.NewRequest("GET", "/", nil))
	if opts.TTL != 2*time.Hour || opts.SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected options: %+v", opts)
	}
}
