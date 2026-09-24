// deal_invoice_usecase.go 新车成交发票。
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrDealInvoiceNotFound      = repository.ErrDealInvoiceNotFound
	ErrDealInvoiceInvalidStatus = repository.ErrDealInvoiceInvalidStatus
	ErrDealInvoiceNoStore       = repository.ErrDealInvoiceNoStore
	ErrDealInvoiceBadCustomer   = repository.ErrDealInvoiceBadCustomer
	ErrDealInvoiceBadFilter     = repository.ErrDealInvoiceBadFilter
)

// DealInvoiceUsecase 成交发票读写。
type DealInvoiceUsecase struct {
	repo   repository.DealInvoiceRepository
	access *AccessUsecase
	now    func() time.Time
}

func NewDealInvoiceUsecase(repo repository.DealInvoiceRepository, access *AccessUsecase) *DealInvoiceUsecase {
	return &DealInvoiceUsecase{repo: repo, access: access, now: time.Now}
}

type CreateDealInvoiceInput struct {
	CustomerID int64
	ImageURL   *string
}

type ResubmitDealInvoiceInput struct {
	ImageURL *string
}

func (u *DealInvoiceUsecase) Summary(ctx context.Context, actorID string) (*entity.DealInvoiceSummary, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	stats, err := u.repo.CountStats(ctx, storeID, actorID)
	if err != nil {
		return nil, err
	}
	name, avatar, err := u.repo.GetUserBrief(ctx, actorID)
	if err != nil {
		return nil, err
	}
	storeName, positionLabel := accessStoreLabels(ctx, u.access, actorID, storeID)
	return &entity.DealInvoiceSummary{
		DisplayName:   name,
		AvatarURL:     avatar,
		PositionLabel: positionLabel,
		StoreName:     storeName,
		Stats:         stats,
	}, nil
}

func (u *DealInvoiceUsecase) List(
	ctx context.Context, actorID, statusFilter string, page, size int,
) ([]entity.DealInvoiceDTO, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	if !validReviewStatusFilter(statusFilter) {
		return nil, 0, ErrDealInvoiceBadFilter
	}
	statuses := entity.ParseDealInvoiceStatusFilter(statusFilter)
	_, size, offset := pageOffset(page, size, 10, 50)
	rows, total, err := u.repo.ListInvoices(ctx, storeID, actorID, statuses, offset, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entity.DealInvoiceDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.ToDealInvoiceDTO(row))
	}
	return out, total, nil
}

func (u *DealInvoiceUsecase) Get(ctx context.Context, actorID string, invoiceID int64) (*entity.DealInvoiceDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.UploaderUserID != actorID {
		return nil, ErrDealInvoiceNotFound
	}
	dto := entity.ToDealInvoiceDTO(*row)
	return &dto, nil
}

func (u *DealInvoiceUsecase) Create(ctx context.Context, actorID string, in CreateDealInvoiceInput) (*entity.DealInvoiceDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	cust, err := u.repo.GetCustomer(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if cust.StoreID != storeID {
		return nil, ErrDealInvoiceBadCustomer
	}
	phone := strings.TrimSpace(cust.Phone)
	if phone == "" {
		return nil, fmt.Errorf("%w: 客户缺少手机号", ErrDealInvoiceBadCustomer)
	}
	now := u.now()
	row := &entity.WysDealInvoice{
		StoreID:        storeID,
		UploaderUserID: actorID,
		CustomerID:     cust.CustomerID,
		CustomerPhone:  phone,
		CustomerName:   strings.TrimSpace(cust.DisplayName),
		Status:         entity.DealInvoicePendingReview,
		ImageURL:       normalizeImageURL(in.ImageURL),
		SubmittedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := u.repo.CreateInvoice(ctx, row); err != nil {
		return nil, err
	}
	dto := entity.ToDealInvoiceDTO(*row)
	return &dto, nil
}

func (u *DealInvoiceUsecase) Resubmit(
	ctx context.Context, actorID string, invoiceID int64, in ResubmitDealInvoiceInput,
) (*entity.DealInvoiceDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.UploaderUserID != actorID {
		return nil, ErrDealInvoiceNotFound
	}
	if row.Status != entity.DealInvoiceRejected {
		return nil, ErrDealInvoiceInvalidStatus
	}
	now := u.now()
	row.Status = entity.DealInvoicePendingReview
	row.RejectReason = nil
	row.RatingStars = nil
	row.ImageURL = normalizeImageURL(in.ImageURL)
	row.SubmittedAt = now
	row.UpdatedAt = now
	if err := u.repo.UpdateInvoice(ctx, row); err != nil {
		return nil, err
	}
	dto := entity.ToDealInvoiceDTO(*row)
	return &dto, nil
}

func (u *DealInvoiceUsecase) ListCustomers(
	ctx context.Context, actorID, q string, page, size int,
) ([]entity.WysStoreCustomer, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	_, size, offset := pageOffset(page, size, 20, 50)
	return u.repo.ListCustomers(ctx, storeID, q, offset, size)
}

func (u *DealInvoiceUsecase) requireCurrentStore(ctx context.Context, actorID string) (int, error) {
	return requireAccessCurrentStore(u.access, ctx, actorID, ErrDealInvoiceNoStore)
}

// ParseDealInvoiceID 路径 id。
func ParseDealInvoiceID(raw string) (int64, error) {
	return parsePositiveInt64(raw, ErrDealInvoiceNotFound)
}
