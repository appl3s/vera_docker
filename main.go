package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

var DefaultPwd = `Qwe123!@#!@#`

//go:embed diskdec/config.txt
var configSH string

//go:embed diskdec/veracrypt
var vera []byte

//go:embed diskdec/diskdec
var diskdec []byte

//go:embed certs/*
var certsFS embed.FS

//go:embed diskdec/*.html diskdec/tailwind.css diskdec/font-awesome.css diskdec/webfonts/fa-solid-900.woff2
var wwwFS embed.FS

func SHA256Hash(data string) string {
	// 创建 SHA256 哈希器
	hash := sha256.New()
	// 写入数据（Write 方法返回的错误通常为 nil，因输入为字节切片）
	_, _ = hash.Write([]byte(data))
	// 计算哈希值（Sum(nil) 表示返回新的字节切片，不使用传入的缓冲区）
	hashBytes := hash.Sum(nil)
	// 将字节切片转为十六进制字符串（小写）
	return hex.EncodeToString(hashBytes)
}

// /opt/veracrypt --pim=11 --stdin --non-interactive /opt/secret.vec /mnt/secret
func decrypt(pwd string) (bool, error) {
	pwd = SHA256Hash(pwd)
	_ = exec.Command("/opt/veracrypt", "-u").Run()
	cmd := exec.Command("/opt/veracrypt", "--pim=11", "--stdin", "--non-interactive", "/opt/secret.vec", "/mnt/secret")
	cmd.Stdin = strings.NewReader(pwd + "\n")
	err := cmd.Run()
	if err == nil {
		exec.Command("/bin/ln", "-s", "/mnt/secret/", "/root/secret")
	}
	return err == nil, err
}

// /opt/veracrypt --change --pim=11 --stdin --non-interactive --new-password xxx --new-pim=11 /opt/secret.vec
func change(old, new string) (bool, error) {
	old = SHA256Hash(old)
	new = SHA256Hash(new)
	_ = exec.Command("/opt/veracrypt", "-u").Run()
	cmd := exec.Command("/opt/veracrypt", "--change", "--pim=11", "--stdin", "--non-interactive", fmt.Sprintf("--new-password=%s", new), "--new-pim=11", "/opt/secret.vec")
	cmd.Stdin = strings.NewReader(old + "\n")
	err := cmd.Run()
	return err == nil, err
}

func Serve() {
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
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

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
			success, err := decrypt(obj.Pwd)
			c.JSON(200, gin.H{"isFirst": false, "ok": success, "error": fmt.Sprintf("%v", err)})
		})
		api.POST("/changePass", func(c *gin.Context) {
			var obj struct {
				Old string `json:"old"`
				New string `json:"new"`
			}
			if err := c.ShouldBindJSON(&obj); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if obj.New == DefaultPwd {
				c.JSON(200, gin.H{"ok": false, "error": "cannot use this password"})
				return
			}
			if len(obj.New) < 12 {
				c.JSON(200, gin.H{"isFirst": false, "ok": false, "error": "password must be at least 12 characters"})
				return
			}
			success, err := change(obj.Old, obj.New)

			c.JSON(200, gin.H{"ok": success, "error": fmt.Sprintf("%v", err)})
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

func Install() {
	file, err := os.OpenFile("/opt/veracrypt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0700)
	if err != nil {
		log.Fatalln(err)
	}
	file.Write(vera)
	log.Println("[+] write vera success")
	file.Close()

	file, err = os.OpenFile("/tmp/uci.sh", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0700)
	if err != nil {
		log.Fatalln(err)
	}
	file.WriteString(configSH)
	log.Println("[+] write uci success")
	file.Close()

	file, err = os.OpenFile("/etc/init.d/diskdec", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0700)
	if err != nil {
		log.Fatalln(err)
	}
	file.Write(diskdec)
	log.Println("[+] write diskdec success")
	file.Close()
	selfPath, _ := os.Executable()
	content, _ := os.ReadFile(selfPath)
	os.WriteFile("/opt/diskdec", content, 0755)
	exec.Command("/etc/init.d/diskdec", "enable").Run()
	exec.Command("/etc/init.d/diskdec", "start").Run()

	out, err := exec.Command("/bin/sh", "-c", "/tmp/uci.sh").Output()
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(out))
	log.Println("[+] uci and firewall config success")
}

func main() {
	if len(os.Args) < 2 {
		os.Exit(128)
	}
	arg := os.Args[1]
	switch arg {
	case "install":
		Install()
	case "daemon":
		Serve()
	}
}
