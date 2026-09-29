// auto_finance.go 购车计算器：平台金融产品与报价读模型。
package entity

import "time"

const (
	WysAutoFinanceProductTable = "wys_auto_finance_product"

	FinanceSubsidyNone      int16 = 0
	FinanceSubsidyRateCut   int16 = 1 // 减年利率（基点）
	FinanceSubsidyAmountFen int16 = 2 // 减贷本金（分）
)

// WysAutoFinanceProduct 平台统一金融产品目录行。
type WysAutoFinanceProduct struct {
	ID                     int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name                   string    `gorm:"column:name;size:128;not null" json:"name"`
	AnnualRateBPS          int       `gorm:"column:annual_rate_bps;not null" json:"annual_rate_bps"` // 450 = 4.50%
	AllowedTermsCSV        string    `gorm:"column:allowed_terms_csv;size:64;not null" json:"allowed_terms_csv"`
	MinDownPaymentBPS      int       `gorm:"column:min_down_payment_bps;not null" json:"min_down_payment_bps"` // 2000 = 20%
	OneTimeFeeFen          int64     `gorm:"column:one_time_fee_fen;not null;default:0" json:"one_time_fee_fen"`
	SubsidyType            int16     `gorm:"column:subsidy_type;not null;default:0" json:"subsidy_type"`
	SubsidyValue           int64     `gorm:"column:subsidy_value;not null;default:0" json:"subsidy_value"` // bps 或 fen
	CompulsoryInsuranceFen int64     `gorm:"column:compulsory_insurance_fen;not null;default:0" json:"compulsory_insurance_fen"`
	CommercialRateBPS      int       `gorm:"column:commercial_rate_bps;not null;default:0" json:"commercial_rate_bps"` // 0=默认不算商业险
	Enabled                bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt              time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (WysAutoFinanceProduct) TableName() string { return WysAutoFinanceProductTable }

// FinanceProductDTO 列表对外读模型（金额为元）。
type FinanceProductDTO struct {
	ID                    int64   `json:"id"`
	Name                  string  `json:"name"`
	AnnualRatePercent     float64 `json:"annual_rate_percent"`
	AllowedTermsMonths    []int   `json:"allowed_terms_months"`
	MinDownPaymentPercent float64 `json:"min_down_payment_percent"`
	OneTimeFee            float64 `json:"one_time_fee"`
	SubsidyType           int16   `json:"subsidy_type"`
	SubsidyRateCutPercent float64 `json:"subsidy_rate_cut_percent,omitempty"`
	SubsidyAmountCut      float64 `json:"subsidy_amount_cut,omitempty"`
	CompulsoryInsurance   float64 `json:"compulsory_insurance"`
	CommercialRatePercent float64 `json:"commercial_rate_percent"`
}

// PurchaseQuoteLine 报价分项（元）。
type PurchaseQuoteLine struct {
	Code   string  `json:"code"`
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
}

// PurchaseQuoteDTO 购车报价结果。
type PurchaseQuoteDTO struct {
	Mode                string              `json:"mode"` // cash | loan
	ProductID           *int64              `json:"product_id,omitempty"`
	ProductName         string              `json:"product_name,omitempty"`
	BarePrice           float64             `json:"bare_price"`
	TaxablePrice        float64             `json:"taxable_price"`
	DownPayment         float64             `json:"down_payment,omitempty"`
	LoanAmount          float64             `json:"loan_amount,omitempty"`
	TermMonths          int                 `json:"term_months,omitempty"`
	EffectiveAnnualRate float64             `json:"effective_annual_rate_percent,omitempty"`
	MonthlyPayment      float64             `json:"monthly_payment,omitempty"`
	TotalInterest       float64             `json:"total_interest,omitempty"`
	TotalRepayment      float64             `json:"total_repayment,omitempty"`
	InitialPayment      float64             `json:"initial_payment"`
	TotalDue            float64             `json:"total_due"`
	Lines               []PurchaseQuoteLine `json:"lines"`
}
