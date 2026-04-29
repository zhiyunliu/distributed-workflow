// Package jwtutil 轻量级 JWT HS256 工具（D4新增）
// 使用标准库实现，不依赖第三方 JWT 包，遵循 go.mod 禁止修改约束。
package jwtutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ErrTokenExpired token 已过期
var ErrTokenExpired = errors.New("token expired")

// ErrTokenInvalid token 非法
var ErrTokenInvalid = errors.New("token invalid")

// Claims JWT payload
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"usr"`
	IssuedAt int64  `json:"iat"`
	ExpireAt int64  `json:"exp"`
}

var header = base64URLEncode([]byte(`{"alg":"HS256","typ":"JWT"}`))

// ResolveSecret 获取有效JWT密钥
// 优先使用 configured，若为空则从环境变量 JWT_SECRET 读取
// 若两者均为空则返回错误（禁止使用空密钥）
func ResolveSecret(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	if env := os.Getenv("JWT_SECRET"); env != "" {
		return env, nil
	}
	return "", errors.New("JWT 密钥未配置：请在配置文件设置 secret 或通过 JWT_SECRET 环境变量提供")
}

// Sign 签发 JWT
func Sign(claims Claims, secret string) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	seg := header + "." + base64URLEncode(payload)
	sig := sign(seg, secret)
	return seg + "." + sig, nil
}

// Parse 解析并验证 JWT
func Parse(token, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	sig := sign(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(sig), []byte(parts[2])) {
		return nil, ErrTokenInvalid
	}

	payload, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, ErrTokenInvalid
	}

	var c Claims
	if err = json.Unmarshal(payload, &c); err != nil {
		return nil, ErrTokenInvalid
	}
	if time.Now().Unix() > c.ExpireAt {
		return nil, ErrTokenExpired
	}
	return &c, nil
}

func sign(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return base64URLEncode(mac.Sum(nil))
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

