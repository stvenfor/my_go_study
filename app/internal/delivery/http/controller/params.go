// params.go controller 层共用的路径参数解析。
//
// parsePathStoreID 原在 access_controller.go（随 access 域切片迁到
// access/handler.go 并变成域私有）。mall_controller 仍留在 controller 包，
// 故在此留一份等价实现；待 mall 域切片后本文件可删。
package controller

import (
	"strconv"

	"app/internal/access"
)

// parsePathStoreID 解析路径里的 store_id；非法时返回 access.ErrInvalidStoreID。
func parsePathStoreID(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, access.ErrInvalidStoreID
	}
	return n, nil
}
