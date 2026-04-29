package types

import "time"

// OAuthPlatform OAuth平台常量
const (
	OAuthPlatformDingTalk   = "dingtalk"   // 钉钉
	OAuthPlatformWechatWork = "wechatwork" // 企业微信
)

// OAuthBinding 第三方平台绑定信息
type OAuthBinding struct {
	ID            int64      `json:"id"`
	SysUserID     string     `json:"sysUserId"`
	Platform      string     `json:"platform"`
	OpenID        string     `json:"openId"`
	UnionID       string     `json:"unionId,omitempty"`
	Nickname      string     `json:"nickname,omitempty"`
	AvatarURL     string     `json:"avatarUrl,omitempty"`
	TokenExpireAt *time.Time `json:"tokenExpireAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// OAuthUserInfo 从OAuth平台获取的用户信息
type OAuthUserInfo struct {
	Platform  string `json:"platform"`
	OpenID    string `json:"openId"`
	UnionID   string `json:"unionId,omitempty"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl,omitempty"`
	Mobile    string `json:"mobile,omitempty"`
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	AppKey    string `json:"appKey"`
	AppSecret string `json:"-"` // 不序列化到JSON
	AgentID   string `json:"agentId"`
}

// WechatWorkConfig 企业微信配置
type WechatWorkConfig struct {
	CorpID      string `json:"corpId"`
	AgentID     string `json:"agentId"`
	AgentSecret string `json:"-"` // 不序列化到JSON
}
