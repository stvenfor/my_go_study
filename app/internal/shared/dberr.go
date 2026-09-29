// dberr.go 跨域共用的数据库错误判别。
package shared

import "strings"

// IsUniqueViolation 判断是否为唯一约束冲突。
//
// 原先散在 postgres 包内（access_repo.go），被 access / home / points /
// after_sales 四个仓储共用；域切片后各域自带实现层，故上提到共享包。
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
