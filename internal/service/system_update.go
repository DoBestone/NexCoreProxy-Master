package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// NexCoreRelease 代理返回的 Release 信息（兼容 GitHub API 格式）
type NexCoreRelease struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	Assets      []NexCoreAsset `json:"assets"`
}

// NexCoreAsset Release 资源文件
type NexCoreAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// NexCoreProxyClient NexCore 代理客户端
type NexCoreProxyClient struct {
	proxyURL  string
	repoToken string
	owner     string
	repo      string
	client    *http.Client
}

// NewNexCoreProxyClient 创建代理客户端
func NewNexCoreProxyClient(proxyURL, repoToken, owner, repo string) *NexCoreProxyClient {
	return &NexCoreProxyClient{
		proxyURL:  proxyURL,
		repoToken: repoToken,
		owner:     owner,
		repo:      repo,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *NexCoreProxyClient) doRequest(method, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/repo/%s/%s%s", c.proxyURL, c.owner, c.repo, path)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.repoToken)
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}

// GetLatestRelease 获取最新 Release 信息
func (c *NexCoreProxyClient) GetLatestRelease() (*NexCoreRelease, error) {
	resp, err := c.doRequest("GET", "/releases/latest")
	if err != nil {
		return nil, fmt.Errorf("请求代理失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("代理返回错误 %d, 读取响应失败: %v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("代理返回错误 %d: %s", resp.StatusCode, string(body))
	}

	var release NexCoreRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析 Release 数据失败: %w", err)
	}
	return &release, nil
}

// ListReleases 获取所有 Release 列表
func (c *NexCoreProxyClient) ListReleases() ([]NexCoreRelease, error) {
	url := fmt.Sprintf("%s/api/repo/%s/%s/releases", c.proxyURL, c.owner, c.repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.repoToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求代理失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("代理返回错误 %d, 读取响应失败: %v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("代理返回错误 %d: %s", resp.StatusCode, string(body))
	}

	var releases []NexCoreRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("解析 Release 列表失败: %w", err)
	}
	return releases, nil
}

// DownloadAsset 通过代理下载 Release Asset
func (c *NexCoreProxyClient) DownloadAsset(downloadURL string) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.repoToken)

	dlClient := &http.Client{Timeout: 10 * time.Minute}
	resp, err := dlClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("下载返回错误 %d", resp.StatusCode)
	}
	return resp.Body, nil
}
