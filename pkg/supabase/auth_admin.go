// auth_admin.go Supabase Auth 管理端辅助（需 service_role）。
package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const adminLookupTimeout = 5 * time.Second

// HasServiceRole 是否配置了 service_role，可用于按邮箱查重。
func (c *Client) HasServiceRole() bool {
	return strings.TrimSpace(c.cfg.ServiceRoleKey) != ""
}

// EmailRegistered 通过 Admin API 判断邮箱是否已注册。
func (c *Client) EmailRegistered(email string) (bool, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return false, fmt.Errorf("email 不能为空")
	}
	if !c.HasServiceRole() {
		return false, fmt.Errorf("未配置 service_role_key")
	}

	ctx, cancel := context.WithTimeout(context.Background(), adminLookupTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimRight(c.cfg.URL, "/")+"/auth/v1/admin/users?page=1&per_page=50",
		nil,
	)
	if err != nil {
		return false, err
	}
	key := c.cfg.ServiceRoleKey
	req.Header.Set("apikey", key)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("查询 Supabase 用户失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取 Supabase 用户响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("查询 Supabase 用户失败: status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Users []struct {
			Email string `json:"email"`
		} `json:"users"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("解析 Supabase 用户响应失败: %w", err)
	}
	for _, user := range payload.Users {
		if strings.EqualFold(strings.TrimSpace(user.Email), email) {
			return true, nil
		}
	}
	return false, nil
}
