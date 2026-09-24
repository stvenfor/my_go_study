// profile_usecase.go Profile 业务用例。
package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

// 约 1.5MB 原始图对应的 base64 上限（字符数）。
const maxAvatarBase64Len = 2_000_000

// ImStoreGroupSyncer 门店群幂等补拉（可选；未接线时 noop）。
type ImStoreGroupSyncer interface {
	EnsureStoreMembership(ctx context.Context, storeID, userID string) (*ImStoreGroupOut, error)
	RemoveStoreMembership(ctx context.Context, storeID, userID string) error
}

// ProfileUsecase 处理用户资料读写。
type ProfileUsecase struct {
	repo           repository.ProfileRepository
	stats          repository.StoreStatsRepository
	storeGroupSync ImStoreGroupSyncer
}

// NewProfileUsecase 创建 Profile 用例。stats 可为 nil（则 stats 全 0）。
func NewProfileUsecase(repo repository.ProfileRepository, stats repository.StoreStatsRepository) *ProfileUsecase {
	return &ProfileUsecase{repo: repo, stats: stats}
}

// SetStoreGroupSync 接线门店群同步（登录切换门店后幂等入群）。
func (u *ProfileUsecase) SetStoreGroupSync(s ImStoreGroupSyncer) {
	if u != nil {
		u.storeGroupSync = s
	}
}

// ErrInvalidStoreID store_id 不是正整数。
var ErrInvalidStoreID = errors.New("store_id 必须为正整数")

// ErrStoreNotFound 店铺不存在或不属于当前用户。
var ErrStoreNotFound = repository.ErrStoreNotFound

// GetProfile 获取当前用户资料（含当前或指定门店统计）。
func (u *ProfileUsecase) GetProfile(ctx context.Context, accessToken, userID, storeID string) (*entity.Profile, entity.UserStoreStats, error) {
	profile, err := u.repo.GetByUserID(ctx, accessToken, userID)
	if err != nil {
		return nil, entity.ZeroUserStoreStats(), err
	}
	sid, err := parseStoreID(storeID)
	if err != nil {
		return nil, entity.ZeroUserStoreStats(), err
	}
	stats := entity.ZeroUserStoreStats()
	if u.stats != nil {
		stats, err = u.stats.Load(ctx, userID, sid)
		if err != nil {
			return nil, entity.ZeroUserStoreStats(), err
		}
	}
	return profile, stats, nil
}

// SwitchStore 切换当前店铺并返回该店铺统计。
func (u *ProfileUsecase) SwitchStore(ctx context.Context, userID string, storeID int) (entity.UserStoreStats, error) {
	if u.stats == nil {
		return entity.ZeroUserStoreStats(), ErrStoreNotFound
	}
	if storeID <= 0 {
		return entity.ZeroUserStoreStats(), ErrInvalidStoreID
	}
	stats, err := u.stats.Switch(ctx, userID, storeID)
	if err != nil {
		if errors.Is(err, repository.ErrStoreNotFound) {
			return entity.ZeroUserStoreStats(), ErrStoreNotFound
		}
		return entity.ZeroUserStoreStats(), err
	}
	// 切换门店成功后幂等拉入该店群；失败不阻断切换。
	if u.storeGroupSync != nil {
		_, _ = u.storeGroupSync.EnsureStoreMembership(ctx, strconv.Itoa(storeID), userID)
	}
	return stats, nil
}

// ListMyStores 当前用户作为成员的经销商列表。
func (u *ProfileUsecase) ListMyStores(ctx context.Context, userID string) ([]entity.UserStoreListItem, error) {
	if u.stats == nil {
		return nil, nil
	}
	return u.stats.ListMyStores(ctx, userID)
}

func parseStoreID(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, ErrInvalidStoreID
	}
	return n, nil
}

// UpdateProfile 更新当前用户资料。
func (u *ProfileUsecase) UpdateProfile(ctx context.Context, accessToken, userID string, input entity.UpdateProfileInput) (*entity.Profile, error) {
	normalized, err := normalizeProfileUpdate(input)
	if err != nil {
		return nil, err
	}
	return u.repo.UpdateByUserID(ctx, accessToken, userID, normalized)
}

func normalizeProfileUpdate(input entity.UpdateProfileInput) (entity.UpdateProfileInput, error) {
	if input.AvatarBase64 == nil || strings.TrimSpace(*input.AvatarBase64) == "" {
		return input, nil
	}

	raw := strings.TrimSpace(*input.AvatarBase64)
	mime := "image/jpeg"
	if idx := strings.Index(raw, ","); idx >= 0 && strings.HasPrefix(strings.ToLower(raw), "data:") {
		header := raw[:idx]
		raw = raw[idx+1:]
		if parts := strings.Split(header, ";"); len(parts) > 0 {
			candidate := strings.TrimPrefix(parts[0], "data:")
			if candidate != "" {
				mime = candidate
			}
		}
	}
	if input.AvatarMime != nil && strings.TrimSpace(*input.AvatarMime) != "" {
		mime = strings.TrimSpace(*input.AvatarMime)
	}
	if !strings.HasPrefix(mime, "image/") {
		return input, fmt.Errorf("avatar_mime 必须为 image/*")
	}
	if len(raw) > maxAvatarBase64Len {
		return input, fmt.Errorf("头像过大，请压缩后重试")
	}
	if _, err := base64.StdEncoding.DecodeString(raw); err != nil {
		// 兼容 URL-safe base64
		if _, err2 := base64.RawStdEncoding.DecodeString(raw); err2 != nil {
			if _, err3 := base64.URLEncoding.DecodeString(raw); err3 != nil {
				return input, fmt.Errorf("avatar_base64 无效")
			}
		}
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, raw)
	input.AvatarURL = &dataURL
	return input, nil
}
