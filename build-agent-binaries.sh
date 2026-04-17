#!/usr/bin/env bash
# 构建 ncp-agent 二进制并放到 Master/binaries/，供 /api/binaries/:name 端点下发
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
AGENT_DIR="$ROOT/Agent"
OUT_DIR="$ROOT/binaries"

mkdir -p "$OUT_DIR"

build() {
  local arch="$1"
  local goarch="$2"
  local out="$OUT_DIR/ncp-agent-linux-$arch"
  echo "→ building $out"
  (cd "$AGENT_DIR" && CGO_ENABLED=0 GOOS=linux GOARCH="$goarch" go build \
    -trimpath -ldflags="-s -w" \
    -o "$out" ./cmd/ncp-agent)
  ls -lh "$out"
}

build amd64 amd64
build arm64 arm64

echo "done. files in $OUT_DIR"
