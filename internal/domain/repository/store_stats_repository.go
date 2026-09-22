// store_stats_repository.go 门店统计仓储。
package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// ErrStoreNotFound 店铺不存在，或当前用户不是该店成员。
var ErrStoreNotFound = errors.New("店铺不存在")

// StoreStatsRepository 读取门店统计并按成员身份切换当前店。
type StoreStatsRepository interface {
	// Load 读取指定店铺的展示数字，职务来自成员。storeID<=0 时优先当前店里他仍是成员的那家，否则取其最小成员店。
	Load(ctx context.Context, userID string, storeID int) (entity.UserStoreStats, error)
	// Switch 仅当他是该店成员时写入 users.current_store_id。
	Switch(ctx context.Context, userID string, storeID int) (entity.UserStoreStats, error)
	// ListMyStores 返回该用户作为成员的门店；is_current 对齐 users.current_store_id。
	ListMyStores(ctx context.Context, userID string) ([]entity.UserStoreListItem, error)
}
