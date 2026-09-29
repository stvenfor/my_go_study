// errors.go 跨域共用的错误哨兵。
package shared

import "errors"

// ErrSearchUnavailable 交易全文搜索（ES）未配置或不可用。
//
// 原先由 app/internal/usecase/transaction_usecase.go 定义，被两处消费：
// transaction 域（Search 返回它、handler 据此回 503）与 ai/cs-agent
// （把「搜索不可用」当良性分支，不向 SSE 抛 error）。批 3 把 transaction
// 切成垂直切片后两处分属不同包，为保证 errors.Is 判定一致，上提到共享包；
// transaction 域内保留 ErrTransactionSearchUnavailable 别名指向同一值。
var ErrSearchUnavailable = errors.New("transaction search unavailable")
