package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"time"

	api "github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/api"
	"github.com/zhiyunliu/distributed-workflow/runtime-execution/pkg/types"
)

// 编译时接口断言
var _ api.OAuthService = (*oauthService)(nil)

type oauthService struct {
	repo api.OAuthRepository
}

// NewOAuthService 创建OAuth服务
func NewOAuthService(repo api.OAuthRepository) api.OAuthService {
	return &oauthService{repo: repo}
}

// GetDingTalkAuthURL 获取钉钉OAuth授权URL
func (s *oauthService) GetDingTalkAuthURL(_ context.Context, redirectURI, state string) (string, error) {
	appKey := os.Getenv("DINGTALK_APP_KEY")
	if appKey == "" {
		return "", fmt.Errorf("DINGTALK_APP_KEY 未配置")
	}
	authURL := fmt.Sprintf(
		"https://login.dingtalk.com/oauth2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid&state=%s",
		appKey,
		neturl.QueryEscape(redirectURI),
		state,
	)
	return authURL, nil
}

// HandleDingTalkCallback 处理钉钉OAuth回调，获取用户信息
func (s *oauthService) HandleDingTalkCallback(_ context.Context, code string) (*types.OAuthUserInfo, error) {
	appKey := os.Getenv("DINGTALK_APP_KEY")
	appSecret := os.Getenv("DINGTALK_APP_SECRET")
	if appKey == "" || appSecret == "" {
		return nil, fmt.Errorf("钉钉应用配置不完整，请检查 DINGTALK_APP_KEY 和 DINGTALK_APP_SECRET")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 1. 用授权码换取用户访问令牌
	tokenReqBody, err := json.Marshal(map[string]string{
		"clientId":     appKey,
		"clientSecret": appSecret,
		"code":         code,
		"grantType":    "authorization_code",
	})
	if err != nil {
		return nil, fmt.Errorf("序列化钉钉token请求体失败: %w", err)
	}

	tokenResp, err := client.Post(
		"https://api.dingtalk.com/v1.0/oauth2/userAccessToken",
		"application/json",
		bytes.NewReader(tokenReqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("请求钉钉userAccessToken失败: %w", err)
	}
	defer tokenResp.Body.Close()

	tokenBody, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取钉钉token响应失败: %w", err)
	}

	var dingTokenResult struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpireIn     int    `json:"expireIn"`
		Code         string `json:"code"`    // 错误码（字符串）
		Message      string `json:"message"` // 错误信息
	}
	if err := json.Unmarshal(tokenBody, &dingTokenResult); err != nil {
		return nil, fmt.Errorf("解析钉钉token响应失败: %w", err)
	}
	if dingTokenResult.AccessToken == "" {
		msg := dingTokenResult.Message
		if msg == "" {
			msg = "未知错误"
		}
		return nil, fmt.Errorf("钉钉获取accessToken失败: %s (code: %s)", msg, dingTokenResult.Code)
	}

	// 2. 用访问令牌获取用户信息
	meReq, err := http.NewRequest(http.MethodGet, "https://api.dingtalk.com/v1.0/contact/users/me", nil)
	if err != nil {
		return nil, fmt.Errorf("创建钉钉获取用户信息请求失败: %w", err)
	}
	meReq.Header.Set("x-acs-dingtalk-access-token", dingTokenResult.AccessToken)

	meResp, err := client.Do(meReq)
	if err != nil {
		return nil, fmt.Errorf("请求钉钉用户信息失败: %w", err)
	}
	defer meResp.Body.Close()

	meBody, err := io.ReadAll(meResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取钉钉用户信息响应失败: %w", err)
	}

	var dingMeResult struct {
		OpenID    string `json:"openId"`
		UnionID   string `json:"unionId"`
		Nick      string `json:"nick"`
		AvatarURL string `json:"avatarUrl"`
		Mobile    string `json:"mobile"`
		Code      string `json:"code"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(meBody, &dingMeResult); err != nil {
		return nil, fmt.Errorf("解析钉钉用户信息失败: %w", err)
	}
	if dingMeResult.OpenID == "" {
		msg := dingMeResult.Message
		if msg == "" {
			msg = "未知错误"
		}
		return nil, fmt.Errorf("钉钉获取用户信息失败: %s", msg)
	}

	return &types.OAuthUserInfo{
		Platform:  types.OAuthPlatformDingTalk,
		OpenID:    dingMeResult.OpenID,
		UnionID:   dingMeResult.UnionID,
		Nickname:  dingMeResult.Nick,
		AvatarURL: dingMeResult.AvatarURL,
		Mobile:    dingMeResult.Mobile,
	}, nil
}

// GetWechatWorkAuthURL 获取企业微信OAuth授权URL
func (s *oauthService) GetWechatWorkAuthURL(_ context.Context, redirectURI, state string) (string, error) {
	corpID := os.Getenv("WECHAT_CORP_ID")
	agentID := os.Getenv("WECHAT_AGENT_ID")
	if corpID == "" || agentID == "" {
		return "", fmt.Errorf("企业微信配置不完整，请检查 WECHAT_CORP_ID 和 WECHAT_AGENT_ID")
	}
	authURL := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_base&state=%s&agentid=%s#wechat_redirect",
		corpID,
		neturl.QueryEscape(redirectURI),
		state,
		agentID,
	)
	return authURL, nil
}

// HandleWechatWorkCallback 处理企业微信OAuth回调，获取用户信息
func (s *oauthService) HandleWechatWorkCallback(_ context.Context, code string) (*types.OAuthUserInfo, error) {
	corpID := os.Getenv("WECHAT_CORP_ID")
	corpSecret := os.Getenv("WECHAT_CORP_SECRET")
	if corpID == "" || corpSecret == "" {
		return nil, fmt.Errorf("企业微信配置不完整，请检查 WECHAT_CORP_ID 和 WECHAT_CORP_SECRET")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	// 1. 获取企业访问令牌
	tokenURL := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		corpID, corpSecret,
	)
	tokenResp, err := client.Get(tokenURL)
	if err != nil {
		return nil, fmt.Errorf("请求企业微信access_token失败: %w", err)
	}
	defer tokenResp.Body.Close()

	tokenBody, err := io.ReadAll(tokenResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取企业微信token响应失败: %w", err)
	}

	var wxTokenResult struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(tokenBody, &wxTokenResult); err != nil {
		return nil, fmt.Errorf("解析企业微信token响应失败: %w", err)
	}
	if wxTokenResult.ErrCode != 0 {
		return nil, fmt.Errorf("企业微信获取access_token失败: %s (errcode: %d)", wxTokenResult.ErrMsg, wxTokenResult.ErrCode)
	}
	if wxTokenResult.AccessToken == "" {
		return nil, fmt.Errorf("企业微信access_token为空")
	}

	// 2. 根据OAuth code获取成员信息
	userInfoURL := fmt.Sprintf(
		"https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo?access_token=%s&code=%s",
		wxTokenResult.AccessToken, code,
	)
	userInfoResp, err := client.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("请求企业微信用户身份失败: %w", err)
	}
	defer userInfoResp.Body.Close()

	userInfoBody, err := io.ReadAll(userInfoResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取企业微信用户身份响应失败: %w", err)
	}

	var wxUserInfoResult struct {
		UserID  string `json:"UserId"`
		OpenID  string `json:"OpenId"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(userInfoBody, &wxUserInfoResult); err != nil {
		return nil, fmt.Errorf("解析企业微信用户身份失败: %w", err)
	}
	if wxUserInfoResult.ErrCode != 0 {
		return nil, fmt.Errorf("企业微信获取用户身份失败: %s (errcode: %d)", wxUserInfoResult.ErrMsg, wxUserInfoResult.ErrCode)
	}

	userInfo := &types.OAuthUserInfo{
		Platform: types.OAuthPlatformWechatWork,
		OpenID:   wxUserInfoResult.OpenID,
	}

	// 3. 若为企业成员，获取用户详情（昵称、头像等）
	if wxUserInfoResult.UserID != "" {
		detailURL := fmt.Sprintf(
			"https://qyapi.weixin.qq.com/cgi-bin/user/get?access_token=%s&userid=%s",
			wxTokenResult.AccessToken, wxUserInfoResult.UserID,
		)
		detailResp, err := client.Get(detailURL)
		if err != nil {
			return nil, fmt.Errorf("请求企业微信用户详情失败: %w", err)
		}
		defer detailResp.Body.Close()

		detailBody, err := io.ReadAll(detailResp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取企业微信用户详情失败: %w", err)
		}

		var wxDetailResult struct {
			UserID  string `json:"userid"`
			Name    string `json:"name"`
			Avatar  string `json:"avatar"`
			Mobile  string `json:"mobile"`
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
		}
		if err := json.Unmarshal(detailBody, &wxDetailResult); err != nil {
			return nil, fmt.Errorf("解析企业微信用户详情失败: %w", err)
		}
		if wxDetailResult.ErrCode == 0 {
			userInfo.Nickname = wxDetailResult.Name
			userInfo.AvatarURL = wxDetailResult.Avatar
			userInfo.Mobile = wxDetailResult.Mobile
		}
	}

	return userInfo, nil
}

// BindUser 将OAuth用户信息绑定到系统用户
func (s *oauthService) BindUser(ctx context.Context, sysUserID string, info *types.OAuthUserInfo) error {
	if sysUserID == "" {
		return fmt.Errorf("系统用户ID不能为空")
	}
	if info == nil {
		return fmt.Errorf("OAuth用户信息不能为空")
	}

	existing, err := s.repo.GetBindingByOpenID(ctx, info.Platform, info.OpenID)
	if err != nil {
		return fmt.Errorf("查询绑定关系失败: %w", err)
	}

	now := time.Now()
	if existing != nil {
		existing.SysUserID = sysUserID
		existing.Nickname = info.Nickname
		existing.AvatarURL = info.AvatarURL
		existing.UnionID = info.UnionID
		existing.UpdatedAt = now
		return s.repo.UpdateBinding(ctx, existing)
	}

	binding := &types.OAuthBinding{
		SysUserID: sysUserID,
		Platform:  info.Platform,
		OpenID:    info.OpenID,
		UnionID:   info.UnionID,
		Nickname:  info.Nickname,
		AvatarURL: info.AvatarURL,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.repo.CreateBinding(ctx, binding)
}

// GetBindingByOpenID 根据平台和OpenID查询绑定关系
func (s *oauthService) GetBindingByOpenID(ctx context.Context, platform, openID string) (*types.OAuthBinding, error) {
	return s.repo.GetBindingByOpenID(ctx, platform, openID)
}

// GetBindingBySysUser 查询系统用户的所有平台绑定
func (s *oauthService) GetBindingBySysUser(ctx context.Context, sysUserID string) ([]*types.OAuthBinding, error) {
	return s.repo.GetBindingBySysUser(ctx, sysUserID)
}


