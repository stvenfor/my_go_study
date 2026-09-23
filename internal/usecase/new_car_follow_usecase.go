// new_car_follow_usecase.go 新车跟进档案。
package usecase

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrNewCarFollowNotFound    = repository.ErrNewCarFollowNotFound
	ErrNewCarFollowNoStore     = repository.ErrNewCarFollowNoStore
	ErrNewCarFollowBadCustomer = repository.ErrNewCarFollowBadCustomer
	ErrNewCarFollowBadLevel    = repository.ErrNewCarFollowBadLevel
	ErrNewCarFollowBadFilter   = repository.ErrNewCarFollowBadFilter
	ErrNewCarFollowBadStage    = repository.ErrNewCarFollowBadStage
	ErrNewCarFollowDuplicate   = repository.ErrNewCarFollowDuplicate
)

// NewCarFollowUsecase 跟进档案读写。
type NewCarFollowUsecase struct {
	repo   repository.NewCarFollowRepository
	access *AccessUsecase
	now    func() time.Time
}

func NewNewCarFollowUsecase(repo repository.NewCarFollowRepository, access *AccessUsecase) *NewCarFollowUsecase {
	return &NewCarFollowUsecase{repo: repo, access: access, now: time.Now}
}

type CreateNewCarFollowInput struct {
	CustomerID      int64
	DisplayName     string
	Phone           string
	FollowLevel     string
	Stage           string
	VehicleInterest string
	BudgetNote      string
	Source          string
	NextFollowUpAt  *time.Time
}

type PatchNewCarFollowInput struct {
	FollowLevel     *string
	Stage           *string
	VehicleInterest *string
	BudgetNote      *string
	Source          *string
	ClosedReason    *string
	NextFollowUpAt  *time.Time
	TouchNextFollow bool // true 时回写客户 next_follow_up_at（可为 nil 清空）
}

func (u *NewCarFollowUsecase) Summary(ctx context.Context, actorID string) (*entity.NewCarFollowSummary, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	stats, err := u.repo.CountStats(ctx, storeID, actorID, u.now())
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
	return &entity.NewCarFollowSummary{
		DisplayName:   name,
		AvatarURL:     avatar,
		PositionLabel: positionLabel,
		StoreName:     storeName,
		Stats:         stats,
	}, nil
}

func (u *NewCarFollowUsecase) List(
	ctx context.Context, actorID string, followLevel, intentBand, stage string, overdue bool, page, size int,
) ([]entity.NewCarFollowFileDTO, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	f := entity.NewCarFollowListFilter{OverdueOnly: overdue}
	if followLevel != "" {
		lv, ok := entity.NormalizeFollowLevel(followLevel)
		if !ok {
			return nil, 0, ErrNewCarFollowBadFilter
		}
		f.FollowLevel = lv
	}
	if intentBand != "" {
		if len(entity.FollowLevelsForIntentBand(intentBand)) == 0 {
			return nil, 0, ErrNewCarFollowBadFilter
		}
		f.IntentBand = intentBand
	}
	if stage != "" {
		st, ok := entity.ParseFollowFileStage(stage)
		if !ok {
			return nil, 0, ErrNewCarFollowBadFilter
		}
		f.Stage = &st
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
	rows, total, err := u.repo.ListFiles(ctx, storeID, actorID, f, u.now(), offset, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entity.NewCarFollowFileDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.ToNewCarFollowFileDTO(row))
	}
	return out, total, nil
}

func (u *NewCarFollowUsecase) Get(ctx context.Context, actorID string, fileID int64) (*entity.NewCarFollowFileDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.OwnerUserID != actorID {
		return nil, ErrNewCarFollowNotFound
	}
	dto := entity.ToNewCarFollowFileDTO(*row)
	return &dto, nil
}

func (u *NewCarFollowUsecase) Create(ctx context.Context, actorID string, in CreateNewCarFollowInput) (*entity.NewCarFollowFileDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	level, ok := entity.NormalizeFollowLevel(in.FollowLevel)
	if !ok {
		return nil, ErrNewCarFollowBadLevel
	}
	stage := entity.FollowStageNew
	if strings.TrimSpace(in.Stage) != "" {
		st, ok := entity.ParseFollowFileStage(in.Stage)
		if !ok {
			return nil, ErrNewCarFollowBadStage
		}
		stage = st
	}

	var cust *entity.WysStoreCustomer
	if in.CustomerID > 0 {
		cust, err = u.repo.GetCustomer(ctx, in.CustomerID)
		if err != nil {
			return nil, err
		}
		if cust.StoreID != storeID {
			return nil, ErrNewCarFollowBadCustomer
		}
	} else {
		name := strings.TrimSpace(in.DisplayName)
		phone := strings.TrimSpace(in.Phone)
		if name == "" || phone == "" {
			return nil, ErrNewCarFollowBadCustomer
		}
		cust = &entity.WysStoreCustomer{
			StoreID:     storeID,
			DisplayName: name,
			Phone:       phone,
		}
		if in.NextFollowUpAt != nil {
			cust.NextFollowUpAt = in.NextFollowUpAt
		}
		if err := u.repo.CreateCustomer(ctx, cust); err != nil {
			return nil, err
		}
	}

	phone := strings.TrimSpace(cust.Phone)
	if phone == "" {
		return nil, ErrNewCarFollowBadCustomer
	}
	if entity.FollowFileIsOpen(stage) {
		existing, err := u.repo.FindOpenFileByCustomer(ctx, storeID, cust.CustomerID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrNewCarFollowDuplicate
		}
	}

	now := u.now()
	row := &entity.WysNewCarFollowFile{
		StoreID:         storeID,
		OwnerUserID:     actorID,
		CustomerID:      cust.CustomerID,
		FollowLevel:     level,
		Stage:           stage,
		VehicleInterest: strings.TrimSpace(in.VehicleInterest),
		BudgetNote:      strings.TrimSpace(in.BudgetNote),
		Source:          strings.TrimSpace(in.Source),
		NextFollowUpAt:  in.NextFollowUpAt,
		CustomerName:    strings.TrimSpace(cust.DisplayName),
		CustomerPhone:   phone,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := u.repo.CreateFile(ctx, row); err != nil {
		return nil, err
	}
	dto := entity.ToNewCarFollowFileDTO(*row)
	return &dto, nil
}

func (u *NewCarFollowUsecase) Patch(
	ctx context.Context, actorID string, fileID int64, in PatchNewCarFollowInput,
) (*entity.NewCarFollowFileDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID || row.OwnerUserID != actorID {
		return nil, ErrNewCarFollowNotFound
	}

	syncFollow := false
	if in.FollowLevel != nil {
		lv, ok := entity.NormalizeFollowLevel(*in.FollowLevel)
		if !ok {
			return nil, ErrNewCarFollowBadLevel
		}
		row.FollowLevel = lv
	}
	if in.Stage != nil {
		st, ok := entity.ParseFollowFileStage(*in.Stage)
		if !ok {
			return nil, ErrNewCarFollowBadStage
		}
		row.Stage = st
	}
	if in.VehicleInterest != nil {
		row.VehicleInterest = strings.TrimSpace(*in.VehicleInterest)
	}
	if in.BudgetNote != nil {
		row.BudgetNote = strings.TrimSpace(*in.BudgetNote)
	}
	if in.Source != nil {
		row.Source = strings.TrimSpace(*in.Source)
	}
	if in.ClosedReason != nil {
		row.ClosedReason = strings.TrimSpace(*in.ClosedReason)
	}
	if in.TouchNextFollow {
		row.NextFollowUpAt = in.NextFollowUpAt
		syncFollow = true
	}
	row.UpdatedAt = u.now()
	if err := u.repo.UpdateFileAndCustomerFollow(ctx, row, syncFollow); err != nil {
		return nil, err
	}
	dto := entity.ToNewCarFollowFileDTO(*row)
	return &dto, nil
}

func (u *NewCarFollowUsecase) ListCustomers(
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

func (u *NewCarFollowUsecase) requireCurrentStore(ctx context.Context, actorID string) (int, error) {
	if u.access == nil {
		return 0, ErrNewCarFollowNoStore
	}
	cur, err := u.access.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if cur == nil || *cur <= 0 {
		return 0, ErrNewCarFollowNoStore
	}
	return *cur, nil
}

// ParseNewCarFollowFileID 路径 id。
func ParseNewCarFollowFileID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrNewCarFollowNotFound
	}
	return id, nil
}
