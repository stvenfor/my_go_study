// used_car_order_usecase.go 二手车业务单。
package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrUsedCarOrderNotFound    = repository.ErrUsedCarOrderNotFound
	ErrUsedCarOrderNoStore     = repository.ErrUsedCarOrderNoStore
	ErrUsedCarOrderBadCustomer = repository.ErrUsedCarOrderBadCustomer
	ErrUsedCarOrderBadFilter   = repository.ErrUsedCarOrderBadFilter
	ErrUsedCarOrderBadInput    = repository.ErrUsedCarOrderBadInput
)

type UsedCarOrderUsecase struct {
	repo   repository.UsedCarOrderRepository
	access *AccessUsecase
	now    func() time.Time
}

func NewUsedCarOrderUsecase(repo repository.UsedCarOrderRepository, access *AccessUsecase) *UsedCarOrderUsecase {
	return &UsedCarOrderUsecase{repo: repo, access: access, now: time.Now}
}

type CreateUsedCarOrderInput struct {
	Kind         string
	CustomerID   int64
	VehicleModel string
	PlateNo      string
	VIN          string
	MileageKm    int
	ModelYear    int
	Amount       float64
	ImageURL     *string
}

func (u *UsedCarOrderUsecase) Summary(ctx context.Context, actorID string) (*entity.UsedCarOrderSummary, error) {
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
	storeName := ""
	positionLabel := ""
	if u.access != nil {
		if store, err := u.access.repo.GetStore(ctx, storeID); err == nil && store != nil {
			storeName = store.Name
		}
		if member, err := u.access.repo.GetMember(ctx, actorID, storeID); err == nil && member != nil {
			positionLabel = entity.StoreRoleLabel(member.Position)
		}
	}
	return &entity.UsedCarOrderSummary{
		DisplayName:   name,
		AvatarURL:     avatar,
		PositionLabel: positionLabel,
		StoreName:     storeName,
		Stats:         stats,
	}, nil
}

func (u *UsedCarOrderUsecase) List(
	ctx context.Context, actorID, statusFilter, kindFilter string, page, size int,
) ([]entity.UsedCarOrderDTO, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	if !validUsedCarStatusFilter(statusFilter) {
		return nil, 0, ErrUsedCarOrderBadFilter
	}
	kind, filterKind, ok := entity.ParseUsedCarKindFilter(kindFilter)
	if !ok {
		return nil, 0, ErrUsedCarOrderBadFilter
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 50 {
		size = 50
	}
	offset := (page - 1) * size
	rows, total, err := u.repo.ListOrders(ctx, storeID, actorID, repository.UsedCarOrderListFilter{
		Statuses:   entity.ParseUsedCarOrderStatusFilter(statusFilter),
		Kind:       kind,
		FilterKind: filterKind,
	}, offset, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entity.UsedCarOrderDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.ToUsedCarOrderDTO(row))
	}
	return out, total, nil
}

func (u *UsedCarOrderUsecase) Get(ctx context.Context, actorID string, orderID int64) (*entity.UsedCarOrderDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.UploaderUserID != actorID {
		return nil, ErrUsedCarOrderNotFound
	}
	dto := entity.ToUsedCarOrderDTO(*row)
	return &dto, nil
}

func (u *UsedCarOrderUsecase) Create(ctx context.Context, actorID string, in CreateUsedCarOrderInput) (*entity.UsedCarOrderDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	kind, ok := entity.ParseUsedCarKind(strings.TrimSpace(in.Kind))
	if !ok {
		return nil, fmt.Errorf("%w: kind", ErrUsedCarOrderBadInput)
	}
	model := strings.TrimSpace(in.VehicleModel)
	plate := strings.TrimSpace(in.PlateNo)
	vin := strings.TrimSpace(in.VIN)
	if model == "" || plate == "" || vin == "" {
		return nil, fmt.Errorf("%w: 车况必填", ErrUsedCarOrderBadInput)
	}
	if in.MileageKm < 0 || in.ModelYear < 1980 || in.ModelYear > 2100 || in.Amount <= 0 {
		return nil, fmt.Errorf("%w: 里程/年款/金额", ErrUsedCarOrderBadInput)
	}
	cust, err := u.repo.GetCustomer(ctx, in.CustomerID)
	if err != nil {
		return nil, err
	}
	if cust.StoreID != storeID {
		return nil, ErrUsedCarOrderBadCustomer
	}
	phone := strings.TrimSpace(cust.Phone)
	if phone == "" {
		return nil, fmt.Errorf("%w: 客户缺少手机号", ErrUsedCarOrderBadCustomer)
	}
	now := u.now()
	row := &entity.WysUsedCarOrder{
		StoreID:        storeID,
		UploaderUserID: actorID,
		Kind:           kind,
		CustomerID:     cust.CustomerID,
		CustomerPhone:  phone,
		CustomerName:   strings.TrimSpace(cust.DisplayName),
		VehicleModel:   model,
		PlateNo:        plate,
		VIN:            vin,
		MileageKm:      in.MileageKm,
		ModelYear:      in.ModelYear,
		Amount:         in.Amount,
		Status:         entity.UsedCarOrderPendingReview,
		ImageURL:       normalizeImageURL(in.ImageURL),
		SubmittedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := u.repo.CreateOrder(ctx, row); err != nil {
		return nil, err
	}
	dto := entity.ToUsedCarOrderDTO(*row)
	return &dto, nil
}

func (u *UsedCarOrderUsecase) ListCustomers(
	ctx context.Context, actorID, q string, page, size int,
) ([]entity.WysStoreCustomer, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	offset := (page - 1) * size
	return u.repo.ListCustomers(ctx, storeID, q, offset, size)
}

func (u *UsedCarOrderUsecase) requireCurrentStore(ctx context.Context, actorID string) (int, error) {
	if u.access == nil {
		return 0, ErrUsedCarOrderNoStore
	}
	cur, err := u.access.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if cur == nil || *cur <= 0 {
		return 0, ErrUsedCarOrderNoStore
	}
	return *cur, nil
}

func validUsedCarStatusFilter(raw string) bool {
	switch strings.TrimSpace(raw) {
	case "", "all", "pending_review", "approved", "rejected":
		return true
	default:
		return false
	}
}

func ParseUsedCarOrderID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrUsedCarOrderNotFound
	}
	return id, nil
}
