package handler

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// findProjectRoot 查找项目根目录
func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	current := wd
	for {
		// 检查当前目录是否包含 go.mod 文件
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// 已到达文件系统根目录
			break
		}
		current = parent
	}

	// 如果在上级目录中没有找到，返回当前工作目录
	return wd
}

// RegisterFrontendRoutes 注册前端静态文件路由
func (s *Server) RegisterFrontendRoutes() {
	distPath := filepath.Join(findProjectRoot(), "config-management", "frontend", "dist")

	// 检查 dist 目录是否存在
	if _, err := os.Stat(distPath); err != nil {
		log.Printf("警告: 未找到前端构建目录 %s，前端页面将不可用", distPath)
		// 即使目录不存在，也注册路由以返回友好的错误信息或404
		s.engine.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(404, gin.H{"code": 404, "message": "API endpoint not found"})
				return
			}
			c.Status(404)
			c.String(404, "前端资源未找到，请先构建前端项目")
		})
		return
	}

	log.Printf("使用前端文件目录: %s", distPath)

	// 处理静态资源 /assets
	assetsPath := filepath.Join(distPath, "assets")
	s.engine.Static("/assets", assetsPath)

	// 添加对 favicon.ico 的特殊处理
	s.engine.GET("/favicon.ico", func(c *gin.Context) {
		faviconPath := filepath.Join(distPath, "favicon.ico")
		if _, err := os.Stat(faviconPath); err == nil {
			c.File(faviconPath)
		} else {
			c.Status(404)
		}
	})

	// 处理 SPA 路由 - 对于不匹配的 API 路由，如果它不是一个 API 请求，则返回 index.html
	s.engine.NoRoute(func(c *gin.Context) {
		// 检查是否是 API 请求
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{"code": 404, "message": "API endpoint not found"})
			return
		}

		// 检查是否是静态资源请求 (除了 /assets 由 Static 处理外，可能还有其他根目录下的静态文件)
		// 这里为了保险起见，如果请求路径对应 dist 下的真实文件，则直接返回
		filePath := filepath.Join(distPath, c.Request.URL.Path)

		// 防止路径遍历攻击
		if !strings.HasPrefix(filepath.Clean(filePath), distPath) {
			c.Status(403)
			return
		}

		if _, err := os.Stat(filePath); err == nil && !os.IsNotExist(err) {
			// 如果是目录，通常不应该直接访问，除非是 index.html 所在的根路径
			// 但 Gin 的 File 方法处理文件比较好，目录交给下面的 index.html 逻辑或者默认行为
			fi, _ := os.Stat(filePath)
			if !fi.IsDir() {
				c.File(filePath)
				return
			}
		}

		// 对于其他请求，返回 index.html 以支持 Vue Router 的 history 模式
		indexPage := filepath.Join(distPath, "index.html")
		if _, err := os.Stat(indexPage); err == nil {
			c.File(indexPage)
		} else {
			c.Status(404)
			c.String(404, "前端资源未找到，请先构建前端项目")
		}
	})
}
