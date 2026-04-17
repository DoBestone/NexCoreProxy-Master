// Package statsclient 通过 `xray api stats` 子进程获取流量统计
//
// 不直接引入 xray-core 库的原因：xray-core 拉满会带来 quic-go / sing / utls 等
// 几十 MB 间接依赖，把 agent 体积撑大。子进程方案唯一缺点是每分钟一次 fork，开销可忽略。
//
// 命令格式（xray 1.8+）:
//
//	xray api stats --server=127.0.0.1:10085 -reset -pattern user
//
// 输出 JSON:
//
//	{ "stat": [ {"name":"user>>>1@nx>>>traffic>>>uplink","value":"12345"}, ... ] }
//
// 用 -reset 让 xray 在返回后清零，下次 Drain 拿到的就是增量。
package statsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"nexcoreproxy-agent/internal/protocol"
)

type Client struct {
	XrayBin   string
	APIServer string // "127.0.0.1:10085"
}

func New(xrayBin string, apiPort int) *Client {
	return &Client{
		XrayBin:   xrayBin,
		APIServer: fmt.Sprintf("127.0.0.1:%d", apiPort),
	}
}

type rawStat struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type rawResp struct {
	Stat []rawStat `json:"stat"`
}

// Drain 实现 pusher.StatsSource 接口
//
// 当前只填 stats（按 email 聚合 up/down）。online 暂返回 nil，待后续从 xray inbound
// 连接表或自定义 access log 解析后填充。
func (c *Client) Drain(ctx context.Context) (map[string]protocol.TrafficDelta, map[string][]string, error) {
	cmd := exec.CommandContext(ctx, c.XrayBin, "api", "stats",
		"--server="+c.APIServer, "-reset", "-pattern", "user")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, nil, fmt.Errorf("xray api stats: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	var resp rawResp
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		// xray 在没数据时可能返回空对象 {} 或纯空，宽容处理
		if len(bytes.TrimSpace(stdout.Bytes())) == 0 {
			return map[string]protocol.TrafficDelta{}, nil, nil
		}
		return nil, nil, fmt.Errorf("decode stats: %w", err)
	}

	result := make(map[string]protocol.TrafficDelta)
	for _, s := range resp.Stat {
		email, dir, ok := parseUserStatName(s.Name)
		if !ok {
			continue
		}
		v, err := strconv.ParseInt(s.Value, 10, 64)
		if err != nil || v == 0 {
			continue
		}
		td := result[email]
		switch dir {
		case "uplink":
			td.Up += v
		case "downlink":
			td.Down += v
		}
		result[email] = td
	}
	return result, nil, nil
}

// parseUserStatName 解析 "user>>>1@nx>>>traffic>>>uplink" → ("1@nx", "uplink", true)
func parseUserStatName(name string) (email, dir string, ok bool) {
	parts := strings.Split(name, ">>>")
	if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
		return "", "", false
	}
	return parts[1], parts[3], true
}
