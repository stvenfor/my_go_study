// purchase_quote_usecase.go 购车报价：金融产品列表 + 全款/贷款权威计算。
package usecase

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrFinanceProductNotFound   = repository.ErrFinanceProductNotFound
	ErrFinanceTermNotAllowed    = errors.New("贷款期数不在产品允许范围")
	ErrFinanceDownPaymentTooLow = errors.New("首付低于产品最低比例")
	ErrFinanceInvalidPrice      = errors.New("价格无效")
	ErrFinanceInvalidMode       = errors.New("报价模式无效")
	ErrFinanceProductRequired   = errors.New("贷款须选择金融产品")
)

const (
	defaultCompulsoryFen int64 = 95000 // 950 元
	defaultCommercialBPS       = 200   // 2%
)

// QuoteInput 购车报价入参（金额单位：元）。
type QuoteInput struct {
	Mode              string
	BarePrice         float64
	TaxablePrice      *float64
	ProductID         int64
	DownPayment       float64
	TermMonths        int
	DisableCommercial bool
}

// PurchaseQuoteUsecase 购车计算器业务。
type PurchaseQuoteUsecase struct {
	repo repository.AutoFinanceRepository
}

func NewPurchaseQuoteUsecase(repo repository.AutoFinanceRepository) *PurchaseQuoteUsecase {
	return &PurchaseQuoteUsecase{repo: repo}
}

func (u *PurchaseQuoteUsecase) ListProducts(ctx context.Context) ([]entity.FinanceProductDTO, error) {
	rows, err := u.repo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.FinanceProductDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toProductDTO(r))
	}
	return out, nil
}

func (u *PurchaseQuoteUsecase) Quote(ctx context.Context, in QuoteInput) (*entity.PurchaseQuoteDTO, error) {
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode != "cash" && mode != "loan" {
		return nil, ErrFinanceInvalidMode
	}
	bareFen := yuanToFen(in.BarePrice)
	if bareFen <= 0 {
		return nil, ErrFinanceInvalidPrice
	}

	var prod *entity.WysAutoFinanceProduct
	if in.ProductID > 0 {
		p, err := u.repo.GetEnabledByID(ctx, in.ProductID)
		if err != nil {
			return nil, err
		}
		prod = p
	}

	taxableFen := divRound(bareFen*100, 113) // / 1.13
	if in.TaxablePrice != nil {
		tf := yuanToFen(*in.TaxablePrice)
		if tf <= 0 {
			return nil, ErrFinanceInvalidPrice
		}
		taxableFen = tf
	}
	taxFen := divRound(taxableFen, 10) // * 10%

	compulsoryFen := defaultCompulsoryFen
	commercialBPS := defaultCommercialBPS
	if prod != nil {
		compulsoryFen = prod.CompulsoryInsuranceFen
		commercialBPS = prod.CommercialRateBPS
	}
	commFen := int64(0)
	if !in.DisableCommercial && commercialBPS > 0 {
		commFen = divRound(bareFen*int64(commercialBPS), 10000)
	}

	if mode == "cash" {
		total := bareFen + taxFen + compulsoryFen + commFen
		lines := []entity.PurchaseQuoteLine{
			{Code: "bare_price", Label: "裸车价", Amount: fenToYuan(bareFen)},
			{Code: "purchase_tax", Label: "购置税", Amount: fenToYuan(taxFen)},
			{Code: "compulsory_insurance", Label: "交强险", Amount: fenToYuan(compulsoryFen)},
		}
		if !in.DisableCommercial {
			lines = append(lines, entity.PurchaseQuoteLine{
				Code: "commercial_insurance", Label: "商业险粗算", Amount: fenToYuan(commFen),
			})
		}
		dto := &entity.PurchaseQuoteDTO{
			Mode:           "cash",
			BarePrice:      fenToYuan(bareFen),
			TaxablePrice:   fenToYuan(taxableFen),
			InitialPayment: fenToYuan(total),
			TotalDue:       fenToYuan(total),
			Lines:          lines,
		}
		if prod != nil {
			id := prod.ID
			dto.ProductID = &id
			dto.ProductName = prod.Name
		}
		return dto, nil
	}

	if prod == nil {
		return nil, ErrFinanceProductRequired
	}
	terms := parseTermsCSV(prod.AllowedTermsCSV)
	if !containsInt(terms, in.TermMonths) {
		return nil, ErrFinanceTermNotAllowed
	}
	downFen := yuanToFen(in.DownPayment)
	minDown := divRound(bareFen*int64(prod.MinDownPaymentBPS), 10000)
	if downFen < minDown {
		return nil, ErrFinanceDownPaymentTooLow
	}
	if downFen >= bareFen {
		return nil, ErrFinanceInvalidPrice
	}
	loanFen := bareFen - downFen
	rateBPS := prod.AnnualRateBPS
	if prod.SubsidyType == entity.FinanceSubsidyRateCut {
		rateBPS -= int(prod.SubsidyValue)
		if rateBPS < 0 {
			rateBPS = 0
		}
	}
	if prod.SubsidyType == entity.FinanceSubsidyAmountFen {
		cut := prod.SubsidyValue
		if cut > loanFen {
			cut = loanFen
		}
		loanFen -= cut
	}
	annual := float64(rateBPS) / 10000
	monthlyFen := equalInstallmentMonthlyFen(loanFen, annual, in.TermMonths)
	totalRepayFen := monthlyFen * int64(in.TermMonths)
	interestFen := totalRepayFen - loanFen
	if interestFen < 0 {
		interestFen = 0
	}

	initialFen := downFen + taxFen + compulsoryFen + commFen + prod.OneTimeFeeFen
	totalDueFen := bareFen + taxFen + compulsoryFen + commFen + prod.OneTimeFeeFen + interestFen

	lines := []entity.PurchaseQuoteLine{
		{Code: "bare_price", Label: "裸车价", Amount: fenToYuan(bareFen)},
		{Code: "down_payment", Label: "首付", Amount: fenToYuan(downFen)},
		{Code: "loan_amount", Label: "贷款额", Amount: fenToYuan(loanFen)},
		{Code: "purchase_tax", Label: "购置税", Amount: fenToYuan(taxFen)},
		{Code: "compulsory_insurance", Label: "交强险", Amount: fenToYuan(compulsoryFen)},
	}
	if !in.DisableCommercial {
		lines = append(lines, entity.PurchaseQuoteLine{
			Code: "commercial_insurance", Label: "商业险粗算", Amount: fenToYuan(commFen),
		})
	}
	if prod.OneTimeFeeFen > 0 {
		lines = append(lines, entity.PurchaseQuoteLine{
			Code: "one_time_fee", Label: "贷款手续费", Amount: fenToYuan(prod.OneTimeFeeFen),
		})
	}
	lines = append(lines,
		entity.PurchaseQuoteLine{Code: "monthly_payment", Label: "月供", Amount: fenToYuan(monthlyFen)},
		entity.PurchaseQuoteLine{Code: "total_interest", Label: "利息合计", Amount: fenToYuan(interestFen)},
	)

	id := prod.ID
	return &entity.PurchaseQuoteDTO{
		Mode:                "loan",
		ProductID:           &id,
		ProductName:         prod.Name,
		BarePrice:           fenToYuan(bareFen),
		TaxablePrice:        fenToYuan(taxableFen),
		DownPayment:         fenToYuan(downFen),
		LoanAmount:          fenToYuan(loanFen),
		TermMonths:          in.TermMonths,
		EffectiveAnnualRate: roundMoney2(float64(rateBPS) / 100),
		MonthlyPayment:      fenToYuan(monthlyFen),
		TotalInterest:       fenToYuan(interestFen),
		TotalRepayment:      fenToYuan(totalRepayFen),
		InitialPayment:      fenToYuan(initialFen),
		TotalDue:            fenToYuan(totalDueFen),
		Lines:               lines,
	}, nil
}

func toProductDTO(r entity.WysAutoFinanceProduct) entity.FinanceProductDTO {
	dto := entity.FinanceProductDTO{
		ID:                    r.ID,
		Name:                  r.Name,
		AnnualRatePercent:     roundMoney2(float64(r.AnnualRateBPS) / 100),
		AllowedTermsMonths:    parseTermsCSV(r.AllowedTermsCSV),
		MinDownPaymentPercent: roundMoney2(float64(r.MinDownPaymentBPS) / 100),
		OneTimeFee:            fenToYuan(r.OneTimeFeeFen),
		SubsidyType:           r.SubsidyType,
		CompulsoryInsurance:   fenToYuan(r.CompulsoryInsuranceFen),
		CommercialRatePercent: roundMoney2(float64(r.CommercialRateBPS) / 100),
	}
	if r.SubsidyType == entity.FinanceSubsidyRateCut {
		dto.SubsidyRateCutPercent = roundMoney2(float64(r.SubsidyValue) / 100)
	}
	if r.SubsidyType == entity.FinanceSubsidyAmountFen {
		dto.SubsidyAmountCut = fenToYuan(r.SubsidyValue)
	}
	return dto
}

func parseTermsCSV(s string) []int {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err == nil && n > 0 {
			out = append(out, n)
		}
	}
	return out
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func yuanToFen(y float64) int64 {
	return int64(math.Round(y * 100))
}

func fenToYuan(f int64) float64 {
	return float64(f) / 100
}

func roundMoney2(v float64) float64 {
	return math.Round(v*100) / 100
}

// divRound 整数除法四舍五入。
func divRound(num, den int64) int64 {
	if den == 0 {
		return 0
	}
	if num >= 0 {
		return (num + den/2) / den
	}
	return (num - den/2) / den
}

func equalInstallmentMonthlyFen(principalFen int64, annualRate float64, months int) int64 {
	if months <= 0 || principalFen <= 0 {
		return 0
	}
	P := float64(principalFen) / 100
	if annualRate <= 0 {
		return yuanToFen(P / float64(months))
	}
	r := annualRate / 12
	pow := math.Pow(1+r, float64(months))
	m := P * r * pow / (pow - 1)
	return yuanToFen(m)
}
