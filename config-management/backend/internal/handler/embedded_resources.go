//go:build production
// +build production

package handler

import "embed"

//go:embed dist/*
var EmbeddedFiles embed.FS