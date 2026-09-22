package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type memFinanceRepo struct {
	rows []entity.WysAutoFinanceProduct
}

func (m *memFinanceRepo) ListEnabled(_ context.Context) ([]entity.WysAutoFinanceProduct, error) {
	var out []entity.WysAutoFinanceProduct
	for _, r := range m.rows {
		if r.Enabled {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memFinanceRepo) GetEnabledByID(_ context.Context, id int64) (*entity.WysAutoFinanceProduct, error) {
	for i := range m.rows {
		if m.rows[i].ID == id && m.rows[i].Enabled {
			cp := m.rows[i]
			return &cp, nil
		}
	}
	return nil, repository.ErrFinanceProductNotFound
}

func seedBankProduct() entity.WysAutoFinanceProduct {
	return entity.WysAutoFinanceProduct{
		ID:                     1,
		Name:                   "示例银行车贷",
		AnnualRateBPS:          450,
		AllowedTermsCSV:        "12,24,36",
		MinDownPaymentBPS:      2000,
		OneTimeFeeFen:          100000, // 1000 元
		SubsidyType:            entity.FinanceSubsidyNone,
		CompulsoryInsuranceFen: 95000,
		CommercialRateBPS:      200, // 2%
		Enabled:                true,
	}
}

func seedOEMProduct() entity.WysAutoFinanceProduct {
	return entity.WysAutoFinanceProduct{
		ID:                     2,
		Name:                   "厂商金融贴息",
		AnnualRateBPS:          450,
		AllowedTermsCSV:        "24,36",
		MinDownPaymentBPS:      2000,
		OneTimeFeeFen:          0,
		SubsidyType:            entity.FinanceSubsidyRateCut,
		SubsidyValue:           50, // 减 0.50%
		CompulsoryInsuranceFen: 95000,
		CommercialRateBPS:      200,
		Enabled:                true,
	}
}

func TestQuoteCashDefaultTaxAndInsurance(t *testing.T) {
	uc := usecase.NewPurchaseQuoteUsecase(&memFinanceRepo{rows: []entity.WysAutoFinanceProduct{seedBankProduct()}})
	q, err := uc.Quote(context.Background(), usecase.QuoteInput{
		Mode:      "cash",
		BarePrice: 100000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if q.TaxablePrice != 88495.58 {
		t.Fatalf("taxable=%v", q.TaxablePrice)
	}
	if q.TotalDue != 111799.56 {
		t.Fatalf("total=%v want 111799.56", q.TotalDue)
	}
	if q.InitialPayment != q.TotalDue {
		t.Fatalf("cash initial should equal total")
	}
}

func TestQuoteCashTaxableOverrideAndDisableCommercial(t *testing.T) {
	uc := usecase.NewPurchaseQuoteUsecase(&memFinanceRepo{rows: []entity.WysAutoFinanceProduct{seedBankProduct()}})
	taxable := 80000.0
	q, err := uc.Quote(context.Background(), usecase.QuoteInput{
		Mode:              "cash",
		BarePrice:         100000,
		TaxablePrice:      &taxable,
		DisableCommercial: true,
		ProductID:         1, // cash may use product for insurance defaults
	})
	if err != nil {
		t.Fatal(err)
	}
	// tax = 80000 * 0.1 = 8000; +950 compulsory; no commercial
	if q.TotalDue != 108950 {
		t.Fatalf("total=%v want 108950", q.TotalDue)
	}
}

func TestQuoteLoanEqualInstallmentWithRateSubsidy(t *testing.T) {
	uc := usecase.NewPurchaseQuoteUsecase(&memFinanceRepo{rows: []entity.WysAutoFinanceProduct{seedOEMProduct()}})
	q, err := uc.Quote(context.Background(), usecase.QuoteInput{
		Mode:        "loan",
		BarePrice:   100000,
		ProductID:   2,
		DownPayment: 30000,
		TermMonths:  36,
	})
	if err != nil {
		t.Fatal(err)
	}
	if q.EffectiveAnnualRate != 4.0 {
		t.Fatalf("rate=%v", q.EffectiveAnnualRate)
	}
	if q.LoanAmount != 70000 {
		t.Fatalf("loan=%v", q.LoanAmount)
	}
	if q.MonthlyPayment != 2066.68 {
		t.Fatalf("monthly=%v want 2066.68", q.MonthlyPayment)
	}
	if q.TotalInterest != 4400.48 {
		t.Fatalf("interest=%v want 4400.48", q.TotalInterest)
	}
	if q.InitialPayment != 41799.56 { // down+tax+comp+comm, fee=0
		t.Fatalf("initial=%v want 41799.56", q.InitialPayment)
	}
}

func TestQuoteLoanOneTimeFee(t *testing.T) {
	uc := usecase.NewPurchaseQuoteUsecase(&memFinanceRepo{rows: []entity.WysAutoFinanceProduct{seedBankProduct()}})
	q, err := uc.Quote(context.Background(), usecase.QuoteInput{
		Mode:        "loan",
		BarePrice:   100000,
		ProductID:   1,
		DownPayment: 30000,
		TermMonths:  36,
	})
	if err != nil {
		t.Fatal(err)
	}
	// no rate subsidy: 4.5% → monthly differs; fee 1000 in initial
	if q.InitialPayment != 42799.56 {
		t.Fatalf("initial=%v want 42799.56", q.InitialPayment)
	}
	if q.EffectiveAnnualRate != 4.5 {
		t.Fatalf("rate=%v", q.EffectiveAnnualRate)
	}
}

func TestQuoteLoanValidation(t *testing.T) {
	uc := usecase.NewPurchaseQuoteUsecase(&memFinanceRepo{rows: []entity.WysAutoFinanceProduct{seedBankProduct()}})
	_, err := uc.Quote(context.Background(), usecase.QuoteInput{
		Mode: "loan", BarePrice: 100000, ProductID: 99, DownPayment: 30000, TermMonths: 36,
	})
	if !errors.Is(err, usecase.ErrFinanceProductNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
	_, err = uc.Quote(context.Background(), usecase.QuoteInput{
		Mode: "loan", BarePrice: 100000, ProductID: 1, DownPayment: 30000, TermMonths: 48,
	})
	if !errors.Is(err, usecase.ErrFinanceTermNotAllowed) {
		t.Fatalf("want term, got %v", err)
	}
	_, err = uc.Quote(context.Background(), usecase.QuoteInput{
		Mode: "loan", BarePrice: 100000, ProductID: 1, DownPayment: 10000, TermMonths: 36,
	})
	if !errors.Is(err, usecase.ErrFinanceDownPaymentTooLow) {
		t.Fatalf("want down, got %v", err)
	}
	_, err = uc.Quote(context.Background(), usecase.QuoteInput{Mode: "cash", BarePrice: 0})
	if !errors.Is(err, usecase.ErrFinanceInvalidPrice) {
		t.Fatalf("want price, got %v", err)
	}
}

func TestListFinanceProducts(t *testing.T) {
	uc := usecase.NewPurchaseQuoteUsecase(&memFinanceRepo{rows: []entity.WysAutoFinanceProduct{
		seedBankProduct(), seedOEMProduct(),
		{ID: 3, Name: "off", Enabled: false},
	}})
	list, err := uc.ListProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d", len(list))
	}
	if list[1].SubsidyRateCutPercent != 0.5 {
		t.Fatalf("subsidy cut=%v", list[1].SubsidyRateCutPercent)
	}
}
