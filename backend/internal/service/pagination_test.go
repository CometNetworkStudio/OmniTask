package service

import (
	"testing"

	"github.com/spf13/viper"
)

func TestNormalizePage(t *testing.T) {
	conf := viper.New()
	// 未配置时用内置默认 100 / 1000
	if page, size := normalizePage(conf, 0, 0); page != 1 || size != 100 {
		t.Fatalf("defaults: got page=%d size=%d", page, size)
	}
	conf.Set("pagination.default", 50)
	conf.Set("pagination.max", 200)
	if page, size := normalizePage(conf, 3, 9999); page != 3 || size != 200 {
		t.Fatalf("clamp: got page=%d size=%d", page, size)
	}
	if page, size := normalizePage(conf, 0, 0); page != 1 || size != 50 {
		t.Fatalf("configured default: got page=%d size=%d", page, size)
	}
}
