package service

import (
	"github.com/spf13/viper"
)

// normalizePage 统一分页：默认与上限来自配置 pagination.default / pagination.max。
func normalizePage(conf *viper.Viper, page, size int) (int, int) {
	def := conf.GetInt("pagination.default")
	if def <= 0 {
		def = 100
	}
	maxSize := conf.GetInt("pagination.max")
	if maxSize <= 0 {
		maxSize = 1000
	}
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = def
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size
}
