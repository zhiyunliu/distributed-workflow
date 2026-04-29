//go:build !production
// +build !production

package handler

import (
	"embed"
)

// 默认为空的嵌入文件
var EmbeddedFiles embed.FS