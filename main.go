package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

var DefaultPwd = `Qwe123!@#!@#`

//go:embed certs/*
var certsFS embed.FS

//go:embed diskdec/*.html diskdec/tailwind.css diskdec/font-awesome.css diskdec/webfonts/fa-solid-900.woff2
var wwwFS embed.FS

func main() {
	// 1. 读取嵌入的证书和私钥
	certData, err := fs.ReadFile(certsFS, "certs/cert.pem")
	if err != nil {
		panic("error1: " + err.Error())
	}
	keyData, err := fs.ReadFile(certsFS, "certs/key.pem")
	if err != nil {
		panic("error2: " + err.Error())
	}

	// 2. 创建临时文件存储证书（因为 RunTLS 需要文件路径）
	// 创建临时目录（程序退出时自动清理）
	tempDir, err := os.MkdirTemp("", "gin-certs")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tempDir) // 程序退出时删除临时目录

	// 写入证书文件
	certPath := filepath.Join(tempDir, "cert.pem")
	if err := os.WriteFile(certPath, certData, 0644); err != nil {
		panic("failed to write cert.pem: " + err.Error())
	}

	// 写入私钥文件（权限更严格，仅当前用户可读写）
	keyPath := filepath.Join(tempDir, "key.pem")
	if err := os.WriteFile(keyPath, keyData, 0600); err != nil {
		panic("failed to write key.pem: " + err.Error())
	}

	// 1. 初始化 Gin 引擎（默认模式，生产环境可改为 gin.ReleaseMode）
	r := gin.Default()
	//gin.SetMode(gin.ReleaseMode)

	api := r.Group("/api")
	{
		api.POST("/decrypt", func(c *gin.Context) {
			var obj struct {
				Pwd string `json:"pwd"`
			}
			if err := c.ShouldBindJSON(&obj); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if obj.Pwd == DefaultPwd {
				c.JSON(200, gin.H{"isFirst": true, "ok": false, "error": "must change password"})
				return
			}
			if len(obj.Pwd) < 12 {
				c.JSON(200, gin.H{"isFirst": false, "ok": false, "error": "password must be at least 12 characters"})
				return
			}
			c.JSON(200, gin.H{"isFirst": false, "ok": true, "error": ""})

		})
		api.POST("/changePass", func(c *gin.Context) {
			var obj struct {
				Pwd string `json:"pwd"`
			}
			if err := c.ShouldBindJSON(&obj); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if obj.Pwd == DefaultPwd {
				c.JSON(200, gin.H{"ok": false, "error": "cannot use this password"})
				return
			}
			if len(obj.Pwd) < 12 {
				c.JSON(200, gin.H{"isFirst": false, "ok": false, "error": "password must be at least 12 characters"})
				return
			}
			c.JSON(200, gin.H{"ok": true, "error": ""})
		})
	}

	// 3. 定义根路径路由（可选）
	fs, err := static.EmbedFolder(wwwFS, "diskdec")
	if err != nil {
		panic(err)
	}
	r.Use(static.Serve("/", fs))

	// 4. 启动服务器，监听 8080 端口
	// 注意：0.0.0.0 表示允许外部访问，仅用于开发环境
	r.RunTLS(":8443", certPath, keyPath) // 传入内存中的证书和私钥
}
