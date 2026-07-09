// client.go 封装 Supabase Go 客户端（anon / service_role / 用户 token）。
package supabase

import (
	"fmt"

	sb "github.com/supabase-community/supabase-go"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// Client 持有 Supabase 多角色客户端。
type Client struct {
	Admin *sb.Client
	Anon  *sb.Client
	cfg   config.SupabaseConfig
}

// New 初始化 Supabase 客户端。
func New(cfg config.SupabaseConfig) (*Client, error) {
	anon, err := sb.NewClient(cfg.URL, cfg.AnonKey, nil)
	if err != nil {
		return nil, fmt.Errorf("初始化 anon 客户端失败: %w", err)
	}

	admin, err := sb.NewClient(cfg.URL, cfg.DBKey(), nil)
	if err != nil {
		return nil, fmt.Errorf("初始化 admin 客户端失败: %w", err)
	}

	return &Client{Admin: admin, Anon: anon, cfg: cfg}, nil
}

// WithUserToken 使用用户 access token 创建客户端（配合 RLS）。
func (c *Client) WithUserToken(accessToken string) (*sb.Client, error) {
	client, err := sb.NewClient(c.cfg.URL, c.cfg.AnonKey, nil)
	if err != nil {
		return nil, err
	}
	client.UpdateAuthSession(accessTokenSession(accessToken))
	return client, nil
}
