// service.go 二手车业务单 —— 域内业务逻辑。
//
// 由 app/internal/usecase/used_car_order_usecase.go 迁入。三处改动：
//  1. UsedCarOrderUsecase → Service（跨域调用方以 used_car.Service 引用）；
//  2. 错误哨兵去 UsedCarOrder 前缀，随 Repository 接口一起下沉 repository.go
//     （域内即命名空间），原先 usecase 层的 `ErrUsedCarOrderX = repository.ErrX` 别名层删除；
//  3. 分页 / 状态白名单 / 图片 URL 归一 / 路径 id 解析改走 app/internal/shared。
package used_car

import (
	"context"
	"fmt"
	"strings"
	"time"

	"app/internal/access"
	"app/internal/shared"
	"model/entity"
)

type Service struct {
	repo      Repository
	accessSvc *access.Service
	now       func() time.Time
}

func NewService(repo Repository, accessSvc *access.Service) *Service {
	return &Service{repo: repo, accessSvc: accessSvc, now: time.Now}
}

type CreateInput struct {
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

func (u *Service) Summary(ctx context.Context, actorID string) (*entity.UsedCarOrderSummary, error) {
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
	return &entity.UsedCarOrderSummary{
		DisplayName:   name,
		AvatarURL:     avatar,
		PositionLabel: positionLabel,
		StoreName:     storeName,
		Stats:         stats,
	}, nil
}

func (u *Service) List(
	ctx context.Context, actorID, statusFilter, kindFilter string, page, size int,
) ([]entity.UsedCarOrderDTO, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	if !shared.ValidReviewStatusFilter(statusFilter) {
		return nil, 0, ErrBadFilter
	}
	kind, filterKind, ok := entity.ParseUsedCarKindFilter(kindFilter)
	if !ok {
		return nil, 0, ErrBadFilter
	}
	_, size, offset := shared.PageOffset(page, size, 10, 50)
	rows, total, err := u.repo.ListOrders(ctx, storeID, actorID, ListFilter{
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

func (u *Service) Get(ctx context.Context, actorID string, orderID int64) (*entity.UsedCarOrderDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.UploaderUserID != actorID {
		return nil, ErrOrderNotFound
	}
	dto := entity.ToUsedCarOrderDTO(*row)
	return &dto, nil
}

func (u *Service) Create(ctx context.Context, actorID string, in CreateInput) (*entity.UsedCarOrderDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	kind, ok := entity.ParseUsedCarKind(strings.TrimSpace(in.Kind))
	if !ok {
		return nil, fmt.Errorf("%w: kind", ErrBadInput)
	}
	model := strings.TrimSpace(in.VehicleModel)
	plate := strings.TrimSpace(in.PlateNo)
	vin := strings.TrimSpace(in.VIN)
	if model == "" || plate == "" || vin == "" {
		return nil, fmt.Errorf("%w: 车况必填", ErrBadInput)
	}
	if in.MileageKm < 0 || in.ModelYear < 1980 || in.ModelYear > 2100 || in.Amount <= 0 {
		return nil, fmt.Errorf("%w: 里程/年款/金额", ErrBadInput)
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
		ImageURL:       shared.NormalizeImageURL(in.ImageURL),
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

func parseOrderID(raw string) (int64, error) {
	return shared.ParsePositiveInt64(raw, ErrOrderNotFound)
}
