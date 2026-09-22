package postgres

import (
	"context"
	"fmt"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

type AutoFinanceRepository struct {
	db *gorm.DB
}

func NewAutoFinanceRepository(db *gorm.DB) *AutoFinanceRepository {
	return &AutoFinanceRepository{db: db}
}

func (r *AutoFinanceRepository) ListEnabled(ctx context.Context) ([]entity.WysAutoFinanceProduct, error) {
	var rows []entity.WysAutoFinanceProduct
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *AutoFinanceRepository) GetEnabledByID(ctx context.Context, id int64) (*entity.WysAutoFinanceProduct, error) {
	var row entity.WysAutoFinanceProduct
	err := r.db.WithContext(ctx).
		Where("id = ? AND enabled = ?", id, true).
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrFinanceProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// EnsureAutoFinanceSchema 建表。
func EnsureAutoFinanceSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.WysAutoFinanceProduct{}); err != nil {
		return fmt.Errorf("migrate wys_auto_finance_product: %w", err)
	}
	return nil
}

// EnsureAutoFinanceSeed 平台目录至少 2 条启用产品（可重复）。
func EnsureAutoFinanceSeed(db *gorm.DB) error {
	if err := EnsureAutoFinanceSchema(db); err != nil {
		return err
	}
	var n int64
	if err := db.Model(&entity.WysAutoFinanceProduct{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	seeds := []entity.WysAutoFinanceProduct{
		{
			Name:                   "示例银行车贷",
			AnnualRateBPS:          450,
			AllowedTermsCSV:        "12,24,36",
			MinDownPaymentBPS:      2000,
			OneTimeFeeFen:          100000,
			SubsidyType:            entity.FinanceSubsidyNone,
			CompulsoryInsuranceFen: 95000,
			CommercialRateBPS:      200,
			Enabled:                true,
		},
		{
			Name:                   "厂商金融贴息",
			AnnualRateBPS:          450,
			AllowedTermsCSV:        "24,36",
			MinDownPaymentBPS:      2000,
			OneTimeFeeFen:          0,
			SubsidyType:            entity.FinanceSubsidyRateCut,
			SubsidyValue:           50,
			CompulsoryInsuranceFen: 95000,
			CommercialRateBPS:      200,
			Enabled:                true,
		},
		{
			Name:                   "低息精品贷",
			AnnualRateBPS:          398,
			AllowedTermsCSV:        "12,24,36,48",
			MinDownPaymentBPS:      1500,
			OneTimeFeeFen:          50000,
			SubsidyType:            entity.FinanceSubsidyAmountFen,
			SubsidyValue:           200000,
			CompulsoryInsuranceFen: 95000,
			CommercialRateBPS:      200,
			Enabled:                true,
		},
	}
	for i := range seeds {
		s := seeds[i]
		if err := db.Create(&s).Error; err != nil {
			return fmt.Errorf("seed finance product: %w", err)
		}
	}
	return nil
}
