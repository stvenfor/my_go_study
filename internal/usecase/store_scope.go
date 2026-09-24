package usecase

import (
	"context"
	"strconv"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// requireAccessCurrentStore 读当前门店；未选中时返回 noStore。
func requireAccessCurrentStore(access *AccessUsecase, ctx context.Context, actorID string, noStore error) (int, error) {
	if access == nil {
		return 0, noStore
	}
	cur, err := access.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if cur == nil || *cur <= 0 {
		return 0, noStore
	}
	return *cur, nil
}

// accessStoreLabels 摘要用门店名与岗位文案（access 缺失时返回空串）。
func accessStoreLabels(ctx context.Context, access *AccessUsecase, actorID string, storeID int) (storeName, positionLabel string) {
	if access == nil {
		return "", ""
	}
	if store, err := access.repo.GetStore(ctx, storeID); err == nil && store != nil {
		storeName = store.Name
	}
	if member, err := access.repo.GetMember(ctx, actorID, storeID); err == nil && member != nil {
		positionLabel = entity.StoreRoleLabel(member.Position)
	}
	return storeName, positionLabel
}

// pageOffset 归一化分页；返回 page、size、offset。
func pageOffset(page, size, defSize, maxSize int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = defSize
	}
	if size > maxSize {
		size = maxSize
	}
	return page, size, (page - 1) * size
}

// validReviewStatusFilter 成交发票 / 二手车业务单共用 status 查询白名单。
func validReviewStatusFilter(raw string) bool {
	switch strings.TrimSpace(raw) {
	case "", "all", "pending_review", "approved", "rejected":
		return true
	default:
		return false
	}
}

func normalizeImageURL(url *string) *string {
	if url == nil {
		return nil
	}
	s := strings.TrimSpace(*url)
	if s == "" {
		return nil
	}
	return &s
}

// parsePositiveInt64 路径/查询正整数；非法时返回 notFound。
func parsePositiveInt64(raw string, notFound error) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, notFound
	}
	return id, nil
}
