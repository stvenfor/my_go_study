// domain_helpers.go 跨域共用的取值 / 分页 / 归一化小工具。
//
// 由 app/internal/usecase/helpers.go 上提而来。该文件原先在 usecase 包内，
// 但 5 个函数只被汽车业务各域（used_car / new_car_follow / after_sales /
// deal_invoice）使用；批 2 把这些域切成垂直切片后，函数跨包不可见，
// 于是按原注释的指向收敛到本共享包并改为导出。
//
// home 域同名私有副本（shanghaiDate）也一并收敛到这里，避免两份实现漂移。
package shared

import (
	"strconv"
	"strings"
	"time"
)

// ShanghaiDate 取「上海时区当天 0 点」，供按天比较的查询使用。
func ShanghaiDate(t time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

// PageOffset 归一化分页；返回 page、size、offset。
func PageOffset(page, size, defSize, maxSize int) (int, int, int) {
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

// ValidReviewStatusFilter 成交发票 / 二手车业务单共用 status 查询白名单。
func ValidReviewStatusFilter(raw string) bool {
	switch strings.TrimSpace(raw) {
	case "", "all", "pending_review", "approved", "rejected":
		return true
	default:
		return false
	}
}

// NormalizeImageURL 空白串归一为 nil，其余去首尾空格。
func NormalizeImageURL(url *string) *string {
	if url == nil {
		return nil
	}
	s := strings.TrimSpace(*url)
	if s == "" {
		return nil
	}
	return &s
}

// ParsePositiveInt64 路径 / 查询正整数；非法时返回调用方给定的 notFound。
func ParsePositiveInt64(raw string, notFound error) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, notFound
	}
	return id, nil
}
