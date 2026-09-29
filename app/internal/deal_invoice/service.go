// service.go 新车成交发票 —— 域内业务逻辑。
//
// 由 app/internal/usecase/deal_invoice_usecase.go 迁入。三处改动：
//  1. DealInvoiceUsecase → Service；CreateDealInvoiceInput → CreateInput 等；
//  2. 错误哨兵去 DealInvoice 前缀并下沉 repository.go；
//  3. 分页 / 状态白名单 / 图片 URL 归一 / 路径 id 解析改走 app/internal/shared。
package deal_invoice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"app/internal/access"
	"app/internal/shared"
	"model/entity"
)

// Service 成交发票读写。
type Service struct {
	repo      Repository
	accessSvc *access.Service
	now       func() time.Time
}

func NewService(repo Repository, accessSvc *access.Service) *Service {
	return &Service{repo: repo, accessSvc: accessSvc, now: time.Now}
}

type CreateInput struct {
	CustomerID int64
	ImageURL   *string
}

type ResubmitInput struct {
	ImageURL *string
}

func (u *Service) Summary(ctx context.Context, actorID string) (*entity.DealInvoiceSummary, error) {
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
	storeName, positionLabel := u.accessSvc.StoreLabels(ctx, actorID, storeID)
	return &entity.DealInvoiceSummary{
		DisplayName:   name,
		AvatarURL:     avatar,
		PositionLabel: positionLabel,
		StoreName:     storeName,
		Stats:         stats,
	}, nil
}

func (u *Service) List(
	ctx context.Context, actorID, statusFilter string, page, size int,
) ([]entity.DealInvoiceDTO, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	if !shared.ValidReviewStatusFilter(statusFilter) {
		return nil, 0, ErrBadFilter
	}
	statuses := entity.ParseDealInvoiceStatusFilter(statusFilter)
	_, size, offset := shared.PageOffset(page, size, 10, 50)
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

func (u *Service) Get(ctx context.Context, actorID string, invoiceID int64) (*entity.DealInvoiceDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetInvoice(ctx, invoiceID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.UploaderUserID != actorID {
		return nil, ErrNotFound
	}
	dto := entity.ToDealInvoiceDTO(*row)
	return &dto, nil
}

func (u *Service) Create(ctx context.Context, actorID string, in CreateInput) (*entity.DealInvoiceDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	cust, err := u.repo.GetCustomer(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if cust.StoreID != storeID {
		return nil, ErrBadCustomer
	}
	phone := strings.TrimSpace(cust.Phone)
	if phone == "" {
		return nil, fmt.Errorf("%w: 客户缺少手机号", ErrBadCustomer)
	}
	now := u.now()
	row := &entity.WysDealInvoice{
		StoreID:        storeID,
		UploaderUserID: actorID,
		CustomerID:     cust.CustomerID,
		CustomerPhone:  phone,
		CustomerName:   strings.TrimSpace(cust.DisplayName),
		Status:         entity.DealInvoicePendingReview,
		ImageURL:       shared.NormalizeImageURL(in.ImageURL),
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

func (u *Service) Resubmit(
	ctx context.Context, actorID string, invoiceID int64, in ResubmitInput,
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
		return nil, ErrNotFound
	}
	if row.Status != entity.DealInvoiceRejected {
		return nil, ErrInvalidStatus
	}
	now := u.now()
	row.Status = entity.DealInvoicePendingReview
	row.RejectReason = nil
	row.RatingStars = nil
	row.ImageURL = shared.NormalizeImageURL(in.ImageURL)
	row.SubmittedAt = now
	row.UpdatedAt = now
	if err := u.repo.UpdateInvoice(ctx, row); err != nil {
		return nil, err
	}
	dto := entity.ToDealInvoiceDTO(*row)
	return &dto, nil
}

func (u *Service) ListCustomers(
	ctx context.Context, actorID, q string, page, size int,
) ([]entity.WysStoreCustomer, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	_, size, offset := shared.PageOffset(page, size, 20, 50)
	return u.repo.ListCustomers(ctx, storeID, q, offset, size)
}

func (u *Service) requireCurrentStore(ctx context.Context, actorID string) (int, error) {
	return u.accessSvc.RequireCurrentStore(ctx, actorID, ErrNoStore)
}

// parseInvoiceID 路径 id。
func parseInvoiceID(raw string) (int64, error) {
	return shared.ParsePositiveInt64(raw, ErrNotFound)
}
