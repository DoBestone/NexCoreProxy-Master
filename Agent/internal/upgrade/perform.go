package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// 防止同时跑多次升级
var upgradeRunning atomic.Bool

// PerformAgentUpgrade 自我升级 ncp-agent
//
// 流程：
//  1. 下载新二进制到 /tmp/ncp-agent.<rand>
//  2. （可选）下载 sha256 文件并校验
//  3. atomic rename 到 /usr/local/bin/ncp-agent.bak（先备份当前），再 rename 新文件到 /usr/local/bin/ncp-agent
//  4. 触发 systemctl restart ncp-agent → systemd 把我们 SIGTERM，新二进制起来
//  5. 失败时 rename .bak 回原位（systemd 会用 .bak 重启，回滚成功）
//
// 因为是自身重启，函数不会"返回成功"——成功路径在我们退出后由新进程接管。
func PerformAgentUpgrade(downloadURLPattern, sha256URL, targetVersion string) {
	if !upgradeRunning.CompareAndSwap(false, true) {
		log.Printf("[upgrade] another upgrade in progress, skip")
		return
	}
	defer upgradeRunning.Store(false)

	url := strings.ReplaceAll(downloadURLPattern, "${ARCH}", goArch())
	log.Printf("[upgrade] downloading agent %s from %s", targetVersion, url)

	tmp, err := downloadToTemp(url, "ncp-agent-new-")
	if err != nil {
		log.Printf("[upgrade] download failed: %v", err)
		return
	}
	defer os.Remove(tmp)

	if sha256URL != "" {
		if err := verifySHA256(tmp, sha256URL); err != nil {
			log.Printf("[upgrade] sha256 mismatch: %v", err)
			return
		}
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		log.Printf("[upgrade] chmod failed: %v", err)
		return
	}

	livePath := "/usr/local/bin/ncp-agent"
	bakPath := livePath + ".bak"
	// 备份当前
	_ = os.Remove(bakPath)
	if err := copyFile(livePath, bakPath); err != nil {
		log.Printf("[upgrade] backup current failed: %v", err)
		return
	}
	if err := os.Rename(tmp, livePath); err != nil {
		log.Printf("[upgrade] swap binary failed: %v, rolling back", err)
		_ = os.Rename(bakPath, livePath)
		return
	}

	log.Printf("[upgrade] agent binary swapped, restarting via systemd")
	if err := exec.Command("systemctl", "restart", "ncp-agent").Run(); err != nil {
		log.Printf("[upgrade] systemctl restart failed: %v, rolling back", err)
		_ = os.Rename(bakPath, livePath)
		_ = exec.Command("systemctl", "restart", "ncp-agent").Run()
	}
	// systemd 会 SIGTERM 我们；这里阻塞等死亡
	time.Sleep(30 * time.Second)
}

// PerformXrayUpgrade 升级 xray-core
//
// 与 agent 升级类似，但更简单：替换二进制 → systemctl restart xray。
// 失败时回滚 .bak，再次 systemctl restart 把流量切回旧版。
func PerformXrayUpgrade(downloadURLPattern, sha256URL, targetVersion string) {
	if !upgradeRunning.CompareAndSwap(false, true) {
		return
	}
	defer upgradeRunning.Store(false)

	url := strings.ReplaceAll(downloadURLPattern, "${ARCH}", goArch())
	log.Printf("[upgrade] downloading xray %s from %s", targetVersion, url)
	tmp, err := downloadToTemp(url, "xray-new-")
	if err != nil {
		log.Printf("[upgrade] xray download failed: %v", err)
		return
	}
	defer os.Remove(tmp)

	if sha256URL != "" {
		if err := verifySHA256(tmp, sha256URL); err != nil {
			log.Printf("[upgrade] xray sha256 mismatch: %v", err)
			return
		}
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return
	}

	livePath := "/usr/local/bin/xray"
	bakPath := livePath + ".bak"
	_ = os.Remove(bakPath)
	if err := copyFile(livePath, bakPath); err != nil {
		log.Printf("[upgrade] xray backup failed: %v", err)
		return
	}
	if err := os.Rename(tmp, livePath); err != nil {
		log.Printf("[upgrade] xray swap failed: %v, rollback", err)
		_ = os.Rename(bakPath, livePath)
		return
	}
	if err := exec.Command("systemctl", "restart", "xray").Run(); err != nil {
		log.Printf("[upgrade] xray restart failed: %v, rollback", err)
		_ = os.Rename(bakPath, livePath)
		_ = exec.Command("systemctl", "restart", "xray").Run()
		return
	}
	log.Printf("[upgrade] xray upgraded to %s", targetVersion)
}

// --- 内部 ---

func goArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}

func downloadToTemp(url, prefix string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.CreateTemp("", prefix+"*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return "", err
	}
	_ = f.Close()
	return f.Name(), nil
}

func verifySHA256(path, sha256URL string) error {
	resp, err := http.Get(sha256URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	expected := strings.Fields(strings.TrimSpace(string(body)))[0] // "<hex>  filename"

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("expected %s got %s", expected, got)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

