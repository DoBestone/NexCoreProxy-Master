package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// ServeBinary GET /api/binaries/:name
//
// 给 install-agent.sh 在节点上拉 ncp-agent 二进制用。
// 严格白名单文件名前缀（防 path traversal + 防泄露其他文件）。
//
// 二进制由运维侧手动构建后放到 BinaryDir：
//
//	cd Agent && GOOS=linux GOARCH=amd64 go build -o $BIN_DIR/ncp-agent-linux-amd64 ./cmd/ncp-agent
//	cd Agent && GOOS=linux GOARCH=arm64 go build -o $BIN_DIR/ncp-agent-linux-arm64 ./cmd/ncp-agent
func (h *Handler) ServeBinary(c *gin.Context) {
	name := c.Param("name")
	// 白名单：只允许 ncp-agent-linux-amd64 / ncp-agent-linux-arm64
	if !strings.HasPrefix(name, "ncp-agent-linux-") {
		c.String(http.StatusForbidden, "forbidden")
		return
	}
	if strings.ContainsAny(name, "/\\") {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	dir := os.Getenv("NCP_BINARY_DIR")
	if dir == "" {
		dir = "binaries" // 相对 Master 工作目录
	}
	full := filepath.Join(dir, name)
	if _, err := os.Stat(full); err != nil {
		c.String(http.StatusNotFound, "binary not found at "+full+
			" (build with: cd Agent && GOOS=linux GOARCH=amd64 go build -o "+full+" ./cmd/ncp-agent)")
		return
	}
	c.FileAttachment(full, name)
}
