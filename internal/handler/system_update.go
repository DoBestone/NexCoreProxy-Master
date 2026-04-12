package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"nexcoreproxy-master/internal/model"
	"nexcoreproxy-master/internal/service"

	"github.com/gin-gonic/gin"
)

// AppVersion 由编译时 -ldflags 注入
var AppVersion = "dev"

const binaryAssetPrefix = "nexcoreproxy-master"

var (
	updateMu           sync.Mutex
	updateConfirmToken string
	updateTokenExpiry  time.Time
)

func newProxyClient() (*service.NexCoreProxyClient, error) {
	var cfg model.NexCoreConfig
	db := model.GetDB()
	if err := db.First(&cfg).Error; err != nil {
		// 不存在则创建默认配置
		cfg = model.NexCoreConfig{
			ProxyURL:  "https://license.nexcores.net",
			Owner:     "DoBestone",
			Repo:      "NexCoreProxy",
			Enabled:   true,
		}
		db.Create(&cfg)
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("NexCore 代理未启用，请在系统设置中配置并启用")
	}
	if cfg.ProxyURL == "" || cfg.RepoToken == "" {
		return nil, fmt.Errorf("未配置 NexCore 代理，请填写代理地址和仓库令牌")
	}
	return service.NewNexCoreProxyClient(cfg.ProxyURL, cfg.RepoToken, cfg.Owner, cfg.Repo), nil
}

// SystemVersion 返回当前版本
func (h *Handler) SystemVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{"version": AppVersion}})
}

// SystemUpdateCheck 检查最新版本
func (h *Handler) SystemUpdateCheck(c *gin.Context) {
	client, err := newProxyClient()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
			"current": AppVersion, "latest": "", "has_update": false, "error": err.Error(),
		}})
		return
	}

	release, err := client.GetLatestRelease()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
			"current": AppVersion, "latest": "", "has_update": false,
			"error": "无法连接 NexCore 代理: " + err.Error(),
		}})
		return
	}

	// 比较版本时忽略 v 前缀 (v1.0.0 == 1.0.0)
	normalizeVer := func(v string) string { return strings.TrimPrefix(v, "v") }
	hasUpdate := normalizeVer(release.TagName) != normalizeVer(AppVersion) && AppVersion != "dev"

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"current":      AppVersion,
		"latest":       release.TagName,
		"has_update":   hasUpdate,
		"changelog":    release.Body,
		"published_at": release.PublishedAt,
	}})
}

// SystemChangelog 获取版本更新日志列表
func (h *Handler) SystemChangelog(c *gin.Context) {
	client, err := newProxyClient()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": err.Error()})
		return
	}

	releases, err := client.ListReleases()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "msg": "获取更新日志失败: " + err.Error()})
		return
	}

	type releaseItem struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		PublishedAt string `json:"published_at"`
	}
	items := make([]releaseItem, 0, len(releases))
	for _, r := range releases {
		items = append(items, releaseItem{
			TagName: r.TagName, Name: r.Name, Body: r.Body, PublishedAt: r.PublishedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{"releases": items}})
}

// SystemUpdatePrepare 生成更新确认令牌（60 秒有效）
func (h *Handler) SystemUpdatePrepare(c *gin.Context) {
	updateMu.Lock()
	defer updateMu.Unlock()

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "msg": "无法生成确认令牌"})
		return
	}
	updateConfirmToken = hex.EncodeToString(b)
	updateTokenExpiry = time.Now().Add(60 * time.Second)

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"confirm_token": updateConfirmToken,
		"message":       "请在 60 秒内确认更新",
	}})
}

// SystemUpdate 执行系统更新
func (h *Handler) SystemUpdate(c *gin.Context) {
	var req struct {
		ConfirmToken string `json:"confirm_token"`
	}
	_ = c.ShouldBindJSON(&req)

	updateMu.Lock()
	valid := updateConfirmToken != "" &&
		req.ConfirmToken == updateConfirmToken &&
		time.Now().Before(updateTokenExpiry)
	updateConfirmToken = ""
	updateTokenExpiry = time.Time{}
	updateMu.Unlock()

	if !valid {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "msg": "确认令牌无效或已过期"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "更新已启动，服务将在下载完成后自动重启"})

	go func() {
		time.Sleep(300 * time.Millisecond)

		wd, err := os.Getwd()
		if err != nil {
			log.Printf("[update] 无法获取工作目录: %v", err)
			return
		}

		// 优先调用 update.sh
		script := filepath.Join(wd, "update.sh")
		if _, err := os.Stat(script); err == nil {
			log.Println("[update] 调用 update.sh --force")
			cmd := exec.Command("bash", script, "--force")
			cmd.Dir = wd
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Printf("[update] update.sh 执行失败: %v，尝试 Go 自更新", err)
				if err2 := goSelfUpdate(); err2 != nil {
					log.Printf("[update] Go 自更新也失败: %v", err2)
				}
			}
			return
		}

		log.Println("[update] update.sh 不存在，使用 Go 自更新")
		if err := goSelfUpdate(); err != nil {
			log.Printf("[update] 自更新失败: %v", err)
		}
	}()
}

// goSelfUpdate 下载当前平台二进制并替换自身
func goSelfUpdate() error {
	client, err := newProxyClient()
	if err != nil {
		return err
	}

	release, err := client.GetLatestRelease()
	if err != nil {
		return fmt.Errorf("获取最新版本失败: %w", err)
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	assetName := fmt.Sprintf("%s-%s-%s", binaryAssetPrefix, goos, goarch)

	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("未找到平台 %s/%s 的预编译二进制 (%s)", goos, goarch, assetName)
	}

	log.Printf("[update] 最新版本 %s，下载 %s", release.TagName, assetName)

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取当前执行文件路径: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("无法解析执行文件路径: %w", err)
	}

	body, err := client.DownloadAsset(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer body.Close()

	tmpFile := exe + ".new"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("无法创建临时文件: %w", err)
	}
	written, err := io.Copy(f, body)
	if err != nil {
		f.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("写入新版本失败: %w", err)
	}
	f.Close()

	const minBinarySize = 1 << 20 // 1MB
	if written < minBinarySize {
		os.Remove(tmpFile)
		return fmt.Errorf("下载的二进制文件异常（%d bytes），已中止更新", written)
	}
	log.Printf("[update] 下载完成: %d bytes", written)

	// 原子替换
	oldFile := exe + ".old"
	if err := os.Rename(exe, oldFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("备份当前二进制失败: %w", err)
	}
	if err := os.Rename(tmpFile, exe); err != nil {
		_ = os.Rename(oldFile, exe)
		os.Remove(tmpFile)
		return fmt.Errorf("替换二进制文件失败: %w", err)
	}
	os.Remove(oldFile)

	// 更新前端
	if err := updateFrontendDist(client, release); err != nil {
		log.Printf("[update] 前端更新失败（后端已更新）: %v", err)
	}

	log.Println("[update] 二进制替换完成，正在重启服务...")
	return syscall.Exec(exe, os.Args, os.Environ())
}

// updateFrontendDist 下载并解压 frontend-dist.tar.gz
func updateFrontendDist(client *service.NexCoreProxyClient, release *service.NexCoreRelease) error {
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == "frontend-dist.tar.gz" {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("release 中未找到 frontend-dist.tar.gz")
	}

	body, err := client.DownloadAsset(downloadURL)
	if err != nil {
		return fmt.Errorf("下载前端包失败: %w", err)
	}
	defer body.Close()

	wd, _ := os.Getwd()
	tmpFile := filepath.Join(wd, "frontend-dist.tar.gz.tmp")
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		os.Remove(tmpFile)
		return err
	}
	f.Close()

	distDir := filepath.Join(wd, "web", "dist")
	os.RemoveAll(distDir)
	os.MkdirAll(distDir, 0755)

	cmd := exec.Command("tar", "-xzf", tmpFile, "-C", distDir)
	if err := cmd.Run(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("解压前端包失败: %w", err)
	}
	os.Remove(tmpFile)
	log.Println("[update] 前端更新完成")
	return nil
}

// SystemProxyConfig 返回当前 NexCore 代理配置
func (h *Handler) SystemProxyConfig(c *gin.Context) {
	var cfg model.NexCoreConfig
	if err := model.GetDB().First(&cfg).Error; err != nil {
		cfg = model.NexCoreConfig{
			ProxyURL: "https://license.nexcores.net",
			Owner:    "DoBestone",
			Repo:     "NexCoreProxy",
			Enabled:  true,
		}
		model.GetDB().Create(&cfg)
	}

	token := cfg.RepoToken
	if len(token) > 8 {
		token = token[:4] + "****" + token[len(token)-4:]
	} else if token != "" {
		token = "****"
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "obj": gin.H{
		"proxy_url":  cfg.ProxyURL,
		"repo_token": token,
		"owner":      cfg.Owner,
		"repo":       cfg.Repo,
		"enabled":    cfg.Enabled,
	}})
}

// SystemProxySaveConfig 保存 NexCore 代理配置
func (h *Handler) SystemProxySaveConfig(c *gin.Context) {
	var req struct {
		ProxyURL  string `json:"proxy_url"`
		RepoToken string `json:"repo_token"`
		Owner     string `json:"owner"`
		Repo      string `json:"repo"`
		Enabled   bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "msg": "参数错误"})
		return
	}

	var cfg model.NexCoreConfig
	if err := model.GetDB().First(&cfg).Error; err != nil {
		cfg = model.NexCoreConfig{}
	}

	cfg.ProxyURL = req.ProxyURL
	if req.RepoToken != "" && !strings.Contains(req.RepoToken, "****") {
		cfg.RepoToken = req.RepoToken
	}
	cfg.Owner = req.Owner
	cfg.Repo = req.Repo
	cfg.Enabled = req.Enabled

	if cfg.ID == 0 {
		model.GetDB().Create(&cfg)
	} else {
		model.GetDB().Save(&cfg)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "msg": "配置已保存"})
}
