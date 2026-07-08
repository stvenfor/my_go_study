package supabase

import (
	"fmt"

	sb "github.com/supabase-community/supabase-go"

	"github.com/stvenfor/my_go_study/internal/config"
)

type Client struct {
	Admin *sb.Client
	Anon  *sb.Client
	cfg   config.SupabaseConfig
}

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

func (c *Client) WithUserToken(accessToken string) (*sb.Client, error) {
	client, err := sb.NewClient(c.cfg.URL, c.cfg.AnonKey, nil)
	if err != nil {
		return nil, err
	}
	client.UpdateAuthSession(accessTokenSession(accessToken))
	return client, nil
}
