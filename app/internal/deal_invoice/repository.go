// repository.go 新车成交发票 —— 域内仓储契约与领域错误。
//
// 由 model/repository/deal_invoice_repository.go 演化而来：接口随域下沉，
// 错误哨兵一并搬入并去掉 DealInvoice 前缀。
package deal_invoice

import (
	"context"
	"errors"

	"model/entity"
)

var (
	ErrNotFound      = errors.New("成交发票不存在")
	ErrInvalidStatus = errors.New("成交发票状态不允许此操作")
	ErrNoStore       = errors.New("未选择当前门店")
	ErrBadCustomer   = errors.New("购车客户无效")
	ErrBadFilter     = errors.New("status 参数无效")
)

// Repository 成交发票 + 选客户读。
type Repository interface {
	CountStats(ctx context.Context, storeID int, uploaderUserID string) (entity.DealInvoiceStats, error)
	ListInvoices(ctx context.Context, storeID int, uploaderUserID string, statuses []int16, offset, limit int) ([]entity.WysDealInvoice, int64, error)
	GetInvoice(ctx context.Context, invoiceID int64) (*entity.WysDealInvoice, error)
	CreateInvoice(ctx context.Context, row *entity.WysDealInvoice) error
	UpdateInvoice(ctx context.Context, row *entity.WysDealInvoice) error

	GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error)
	ListCustomers(ctx context.Context, storeID int, q string, offset, limit int) ([]entity.WysStoreCustomer, int64, error)

	GetUserBrief(ctx context.Context, userID string) (userName, avatarURL string, err error)
}
