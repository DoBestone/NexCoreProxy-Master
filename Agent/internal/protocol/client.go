package protocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client 是 Master HTTP 协议客户端
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient masterURL 不带尾斜杠，token = node_token
func NewClient(masterURL, token string) *Client {
	return &Client{
		baseURL: masterURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchConfig GET /api/v1/server/config?etag=...
//
// 三种情况：
//   - 200 + cfg：配置已更新，调用方应渲染并 reload xray
//   - notModified=true：etag 匹配，无需更新
//   - error：网络/服务端错误，调用方按重试或回落缓存
func (c *Client) FetchConfig(ctx context.Context, currentEtag string) (cfg *ServerConfig, notModified bool, err error) {
	u := c.baseURL + "/api/v1/server/config"
	if currentEtag != "" {
		u += "?etag=" + url.QueryEscape(currentEtag)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	c.applyAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, true, nil
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, false, err
		}
		var out ServerConfig
		if err := json.Unmarshal(body, &out); err != nil {
			return nil, false, fmt.Errorf("decode config: %w", err)
		}
		return &out, false, nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("master returned %s: %s", resp.Status, string(body))
	}
}

// PushReport POST /api/v1/server/push
func (c *Client) PushReport(ctx context.Context, payload *PushRequest) (*PushResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/server/push", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("master returned %s: %s", resp.Status, string(respBody))
	}
	var out PushResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode push resp: %w", err)
	}
	return &out, nil
}

func (c *Client) applyAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	// 留个 hint 给服务端日志便于排查
	req.Header.Set("X-NCP-Agent", "ncp-agent/0.1")
}

