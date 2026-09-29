// ANNOTATED_BY_SCRIPT:annotate_go_every_line.py
// 声明当前 Go 包名为 req
package req

// 定义结构体类型 PageInfo
type PageInfo struct {
	// 语句或表达式
	Page int `json:"page" form:"page"`
	// 语句或表达式
	PageSize int `json:"pageSize" form:"pageSize"`
	// 语句或表达式
}
