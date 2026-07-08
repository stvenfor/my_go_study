package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain"
	"github.com/stvenfor/my_go_study/internal/supabase"
)

type ProfileRepository struct {
	supabase *supabase.Client
}

func NewProfileRepository(client *supabase.Client) *ProfileRepository {
	return &ProfileRepository{supabase: client}
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, accessToken, userID string) (*domain.Profile, error) {
	client, err := r.supabase.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

	data, _, err := client.From(domain.ProfilesTable).
		Select("*", "", false).
		Eq("id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("查询 profile 失败: %w", err)
	}

	var rows []domain.Profile
	if err := json.Unmarshal(data, &rows); err == nil && len(rows) > 0 {
		return &rows[0], nil
	}

	var profile domain.Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("解析 profile 失败: %w", err)
	}
	return &profile, nil
}

func (r *ProfileRepository) UpdateByUserID(ctx context.Context, accessToken, userID string, input domain.UpdateProfileInput) (*domain.Profile, error) {
	client, err := r.supabase.WithUserToken(accessToken)
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

	data, _, err := client.From(domain.ProfilesTable).
		Update(payload, "representation", "").
		Eq("id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("更新 profile 失败: %w", err)
	}

	var profile domain.Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		var rows []domain.Profile
		if err2 := json.Unmarshal(data, &rows); err2 == nil && len(rows) > 0 {
			return &rows[0], nil
		}
		return nil, fmt.Errorf("解析 profile 失败: %w", err)
	}
	return &profile, nil
}
