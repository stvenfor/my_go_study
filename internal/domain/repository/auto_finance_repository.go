// auto_finance_repository.go 购车计算器金融产品仓储。
package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrFinanceProductNotFound = errors.New("金融产品不存在")
)

// AutoFinanceRepository 平台金融产品读。
type AutoFinanceRepository interface {
	ListEnabled(ctx context.Context) ([]entity.WysAutoFinanceProduct, error)
	GetEnabledByID(ctx context.Context, id int64) (*entity.WysAutoFinanceProduct, error)
}
