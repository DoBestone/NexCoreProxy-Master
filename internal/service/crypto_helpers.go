package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// randomUUID 标准 v4 UUID（不引 google/uuid 是为了避免又拉一个依赖）
func randomUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// randomToken 生成 hex 编码的随机串（默认场景：trunk password、ss psk 等）
func randomToken(byteLen int) string {
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// genRealityKeys 生成 X25519 keypair + 8 字节 shortID
//
// 与 `xray x25519` 输出格式对齐（base64 url-encoded, no padding）。
// 调用 xray 子进程也能拿到，但我们这里直接算，避免依赖节点上有 xray 二进制。
func genRealityKeys() (priv, pub, shortID string, err error) {
	var rawPriv [32]byte
	if _, err = rand.Read(rawPriv[:]); err != nil {
		return
	}
	// curve25519 私钥规约
	rawPriv[0] &= 248
	rawPriv[31] &= 127
	rawPriv[31] |= 64

	rawPub, err := curve25519.X25519(rawPriv[:], curve25519.Basepoint)
	if err != nil {
		return
	}
	priv = base64.RawURLEncoding.EncodeToString(rawPriv[:])
	pub = base64.RawURLEncoding.EncodeToString(rawPub)

	var rawShort [8]byte
	if _, err = rand.Read(rawShort[:]); err != nil {
		return
	}
	shortID = hex.EncodeToString(rawShort[:])
	return
}

// randomSubscribeToken 给用户订阅链接用的高熵 token
func randomSubscribeToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// randomSS2022PSK 生成符合 SS-2022 规范的 PSK
//
// SS-2022 的 password 字段要求是 base64 编码的原始密钥字节：
//   - AES-128-GCM → 16 bytes → base64 22 chars (padded 24)
//   - AES-256-GCM → 32 bytes → base64 44 chars
//   - chacha20-poly1305 → 32 bytes → base64 44 chars
//
// 不能用 hex 字符串当 PSK — xray 会用 base64 解码得到乱字节，与客户端不匹配导致 AEAD 失败。
func randomSS2022PSK(bytes int) string {
	b := make([]byte, bytes)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
