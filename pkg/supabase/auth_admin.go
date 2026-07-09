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

	"github.com/google/uuid"
	"github.com/supabase-community/gotrue-go/types"
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

	users, err := c.listAdminUsers()
	if err != nil {
		return false, err
	}
	for _, user := range users {
		if strings.EqualFold(strings.TrimSpace(user.Email), email) {
			return true, nil
		}
	}
	return false, nil
}

// PhoneRegistered 通过 Admin API 按手机号查找用户，返回 user ID。
func (c *Client) PhoneRegistered(e164 string) (string, bool, error) {
	e164 = strings.TrimSpace(e164)
	if e164 == "" {
		return "", false, fmt.Errorf("phone 不能为空")
	}
	if !c.HasServiceRole() {
		return "", false, fmt.Errorf("未配置 service_role_key")
	}

	users, err := c.listAdminUsers()
	if err != nil {
		return "", false, err
	}
	for _, user := range users {
		if strings.TrimSpace(user.Phone) == e164 {
			return user.ID.String(), true, nil
		}
	}
	return "", false, nil
}

// EnsureDevPhoneUser 确保测试手机号用户在 Supabase 存在（含映射邮箱与密码）。
func (c *Client) EnsureDevPhoneUser(e164, devEmail, password, displayName string) (string, error) {
	if !c.HasServiceRole() {
		return "", fmt.Errorf("未配置 service_role_key")
	}
	e164 = strings.TrimSpace(e164)
	devEmail = strings.TrimSpace(strings.ToLower(devEmail))
	password = strings.TrimSpace(password)
	if e164 == "" || devEmail == "" || password == "" {
		return "", fmt.Errorf("phone/email/password 不能为空")
	}

	if userID, ok, err := c.PhoneRegistered(e164); err != nil {
		return "", err
	} else if ok {
		if err := c.ensureDevPhoneUserCredentials(userID, devEmail, password, displayName); err != nil {
			return "", err
		}
		return userID, nil
	}

	metadata := map[string]interface{}{}
	if name := strings.TrimSpace(displayName); name != "" {
		metadata["display_name"] = name
	}
	pwd := password
	resp, err := c.Admin.Auth.AdminCreateUser(types.AdminCreateUserRequest{
		Email:        devEmail,
		Phone:        e164,
		Password:     &pwd,
		EmailConfirm: true,
		PhoneConfirm: true,
		UserMetadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("创建测试手机号用户失败: %w", err)
	}
	return resp.User.ID.String(), nil
}

func (c *Client) ensureDevPhoneUserCredentials(userID, devEmail, password, displayName string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("无效 user id: %w", err)
	}
	metadata := map[string]interface{}{}
	if name := strings.TrimSpace(displayName); name != "" {
		metadata["display_name"] = name
	}
	_, err = c.Admin.Auth.AdminUpdateUser(types.AdminUpdateUserRequest{
		UserID:       uid,
		Email:        devEmail,
		Password:     password,
		EmailConfirm: true,
		PhoneConfirm: true,
		UserMetadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("更新测试手机号用户失败: %w", err)
	}
	return nil
}

func (c *Client) listAdminUsers() ([]types.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), adminLookupTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimRight(c.cfg.URL, "/")+"/auth/v1/admin/users?page=1&per_page=200",
		nil,
	)
	if err != nil {
		return nil, err
	}
	key := c.cfg.ServiceRoleKey
	req.Header.Set("apikey", key)
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询 Supabase 用户失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Supabase 用户响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("查询 Supabase 用户失败: status %d: %s", resp.StatusCode, body)
	}

	var payload struct {
		Users []types.User `json:"users"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析 Supabase 用户响应失败: %w", err)
	}
	return payload.Users, nil
}
