// deal_invoice_repository.go 新车成交发票仓储。
package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrDealInvoiceNotFound      = errors.New("成交发票不存在")
	ErrDealInvoiceInvalidStatus = errors.New("成交发票状态不允许此操作")
	ErrDealInvoiceNoStore       = errors.New("未选择当前门店")
	ErrDealInvoiceBadCustomer   = errors.New("购车客户无效")
	ErrDealInvoiceBadFilter     = errors.New("status 参数无效")
)

// DealInvoiceRepository 成交发票 + 选客户读。
type DealInvoiceRepository interface {
	CountStats(ctx context.Context, storeID int, uploaderUserID string) (entity.DealInvoiceStats, error)
	ListInvoices(ctx context.Context, storeID int, uploaderUserID string, statuses []int16, offset, limit int) ([]entity.WysDealInvoice, int64, error)
	GetInvoice(ctx context.Context, invoiceID int64) (*entity.WysDealInvoice, error)
	CreateInvoice(ctx context.Context, row *entity.WysDealInvoice) error
	UpdateInvoice(ctx context.Context, row *entity.WysDealInvoice) error

	GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error)
	ListCustomers(ctx context.Context, storeID int, q string, offset, limit int) ([]entity.WysStoreCustomer, int64, error)

	GetUserBrief(ctx context.Context, userID string) (userName, avatarURL string, err error)
}
