// helpers.go 未迁移域共用的取值/分页小工具。
//
// 由原 store_scope.go 演化而来：其中 access 专属的两个函数
// （requireAccessCurrentStore / accessStoreLabels）已随 access 域切片迁走，
// 成为 access.Service 上的 RequireCurrentStore / StoreLabels 方法。
//
// 本文件剩下的工具目前只被汽车业务各域使用（deal_invoice / new_car_follow /
// after_sales / used_car），它们尚未切片，故暂留 usecase 包；待批 2 迁完后
// 本文件应随之消失或收敛进 app/internal/shared。
package usecase

import (
	"strconv"
	"strings"
	"time"
)

// shanghaiDate 取「上海时区当天 0 点」，供按天比较的查询使用。
// 原在 home_todo_usecase.go，随 home 域切片迁走后此处留一份同实现副本
// （unqualified 同名函数不再跨包可见）。
func shanghaiDate(t time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
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
