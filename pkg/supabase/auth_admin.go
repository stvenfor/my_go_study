// auth_admin.go Supabase Auth 管理端辅助（需 service_role）。
package supabase

import (
	"fmt"
	"strings"
)

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

	resp, err := c.Admin.Auth.AdminListUsers()
	if err != nil {
		return false, fmt.Errorf("查询 Supabase 用户失败: %w", err)
	}
	for _, user := range resp.Users {
		if strings.EqualFold(strings.TrimSpace(user.Email), email) {
			return true, nil
		}
	}
	return false, nil
}
