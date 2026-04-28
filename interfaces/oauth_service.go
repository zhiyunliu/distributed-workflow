package interfaces

import (
	"context"

	"github.com/zhiyunliu/distributed-workflow/types"
)

// OAuthService 第三方平台OAuth服务接口
type OAuthService interface {
	// GetDingTalkAuthURL 获取钉钉OAuth授权URL
	GetDingTalkAuthURL(ctx context.Context, redirectURI, state string) (string, error)
	// HandleDingTalkCallback 处理钉钉OAuth回调，获取用户信息
	HandleDingTalkCallback(ctx context.Context, code string) (*types.OAuthUserInfo, error)
	// GetWechatWorkAuthURL 获取企业微信OAuth授权URL
	GetWechatWorkAuthURL(ctx context.Context, redirectURI, state string) (string, error)
	// HandleWechatWorkCallback 处理企业微信OAuth回调，获取用户信息
	HandleWechatWorkCallback(ctx context.Context, code string) (*types.OAuthUserInfo, error)
	// BindUser 将OAuth用户信息绑定到系统用户
	BindUser(ctx context.Context, sysUserID string, info *types.OAuthUserInfo) error
	// GetBindingByOpenID 根据平台和OpenID查询绑定关系
	GetBindingByOpenID(ctx context.Context, platform, openID string) (*types.OAuthBinding, error)
	// GetBindingBySysUser 查询系统用户的所有平台绑定
	GetBindingBySysUser(ctx context.Context, sysUserID string) ([]*types.OAuthBinding, error)
}

// OAuthRepository OAuth绑定数据仓库接口
type OAuthRepository interface {
	CreateBinding(ctx context.Context, binding *types.OAuthBinding) error
	GetBindingByOpenID(ctx context.Context, platform, openID string) (*types.OAuthBinding, error)
	GetBindingBySysUser(ctx context.Context, sysUserID string) ([]*types.OAuthBinding, error)
	UpdateBinding(ctx context.Context, binding *types.OAuthBinding) error
}
