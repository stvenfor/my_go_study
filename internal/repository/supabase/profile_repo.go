// profile_repo.go 通过 Supabase PostgREST 访问 profiles 表。
package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
)

// ProfileRepository profiles 仓储实现。
type ProfileRepository struct {
	client *pkgsb.Client
}

// NewProfileRepository 创建 profiles 仓储。
func NewProfileRepository(client *pkgsb.Client) *ProfileRepository {
	return &ProfileRepository{client: client}
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, accessToken, userID string) (*entity.Profile, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

	data, _, err := client.From(entity.ProfilesTable).
		Select("*", "", false).
		Eq("id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("查询 profile 失败: %w", err)
	}

	return unmarshalSingle[entity.Profile](data)
}

func (r *ProfileRepository) UpdateByUserID(ctx context.Context, accessToken, userID string, input entity.UpdateProfileInput) (*entity.Profile, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if input.DisplayName != nil {
		payload["display_name"] = *input.DisplayName
	}
	if input.AvatarURL != nil {
		payload["avatar_url"] = *input.AvatarURL
	}

	data, _, err := client.From(entity.ProfilesTable).
		Update(payload, "representation", "").
		Eq("id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("更新 profile 失败: %w", err)
	}

	return unmarshalSingle[entity.Profile](data)
}

func unmarshalSingle[T any](data []byte) (*T, error) {
	var item T
	if err := json.Unmarshal(data, &item); err == nil {
		return &item, nil
	}

	var rows []T
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("未找到记录")
	}
	return &rows[0], nil
}
