// repository.go 购车计算器金融产品 —— 域内仓储契约与领域错误。
//
// 由 model/repository/auto_finance_repository.go 演化而来：接口随域下沉，
// 错误哨兵一并搬入并去掉 Finance 前缀。
package purchase_calculator

import (
	"context"
	"errors"

	"model/entity"
)

var (
	ErrProductNotFound = errors.New("金融产品不存在")
	// 以下五条原先直接定义在 usecase 层（非 repository 别名），随域一并下沉。
	ErrTermNotAllowed    = errors.New("贷款期数不在产品允许范围")
	ErrDownPaymentTooLow = errors.New("首付低于产品最低比例")
	ErrInvalidPrice      = errors.New("价格无效")
	ErrInvalidMode       = errors.New("报价模式无效")
	ErrProductRequired   = errors.New("贷款须选择金融产品")
)

// Repository 平台金融产品读。
type Repository interface {
	ListEnabled(ctx context.Context) ([]entity.WysAutoFinanceProduct, error)
	GetEnabledByID(ctx context.Context, id int64) (*entity.WysAutoFinanceProduct, error)
}
