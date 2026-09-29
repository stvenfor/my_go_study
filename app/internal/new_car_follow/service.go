// service.go 新车跟进档案 —— 域内业务逻辑。
//
// 由 app/internal/usecase/new_car_follow_usecase.go 迁入。三处改动：
//  1. NewCarFollowUsecase → Service；NewNewCarFollowUsecase → NewService；
//  2. 错误哨兵去 NewCarFollow 前缀并下沉 repository.go；
//  3. 分页 / 路径 id 解析改走 app/internal/shared。
package new_car_follow

import (
	"context"
	"strings"
	"time"

	"app/internal/access"
	"app/internal/shared"
	"model/entity"
)

const followStoreAdminPerm = entity.PermRoleAssignStore

// Service 跟进档案读写。
type Service struct {
	repo      Repository
	accessSvc *access.Service
	now       func() time.Time
}

func NewService(repo Repository, accessSvc *access.Service) *Service {
	return &Service{repo: repo, accessSvc: accessSvc, now: time.Now}
}

type CreateInput struct {
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

type PatchInput struct {
	FollowLevel     *string
	Stage           *string
	VehicleInterest *string
	BudgetNote      *string
	Source          *string
	ClosedReason    *string
	NextFollowUpAt  *time.Time
	TouchNextFollow bool // true 时回写客户 next_follow_up_at（可为 nil 清空）
}

type CreateFollowLogInput struct {
	Body           string
	FollowLevel    string     // 可选；空=不改级别
	NextFollowUpAt *time.Time // 非 nil 时更新档案+客户；传指针且需 TouchNext
	TouchNext      bool
}

func (u *Service) Summary(ctx context.Context, actorID string) (*entity.NewCarFollowSummary, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	ownerFilter, err := u.ownerScope(ctx, actorID, storeID)
	if err != nil {
		return nil, err
	}
	stats, err := u.repo.CountStats(ctx, storeID, ownerFilter, u.now())
	if err != nil {
		return nil, err
	}
	name, avatar, err := u.repo.GetUserBrief(ctx, actorID)
	if err != nil {
		return nil, err
	}
	storeName, positionLabel := u.accessSvc.StoreLabels(ctx, actorID, storeID)
	return &entity.NewCarFollowSummary{
		DisplayName:   name,
		AvatarURL:     avatar,
		PositionLabel: positionLabel,
		StoreName:     storeName,
		Stats:         stats,
	}, nil
}

func (u *Service) List(
	ctx context.Context, actorID string, followLevel, intentBand, stage string, overdue bool, page, size int,
) ([]entity.NewCarFollowFileDTO, int64, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, err
	}
	ownerFilter, err := u.ownerScope(ctx, actorID, storeID)
	if err != nil {
		return nil, 0, err
	}
	f := entity.NewCarFollowListFilter{OverdueOnly: overdue}
	if followLevel != "" {
		lv, ok := entity.NormalizeFollowLevel(followLevel)
		if !ok {
			return nil, 0, ErrBadFilter
		}
		f.FollowLevel = lv
	}
	if intentBand != "" {
		if len(entity.FollowLevelsForIntentBand(intentBand)) == 0 {
			return nil, 0, ErrBadFilter
		}
		f.IntentBand = intentBand
	}
	if stage != "" {
		st, ok := entity.ParseFollowFileStage(stage)
		if !ok {
			return nil, 0, ErrBadFilter
		}
		f.Stage = &st
	}
	_, size, offset := shared.PageOffset(page, size, 10, 50)
	rows, total, err := u.repo.ListFiles(ctx, storeID, ownerFilter, f, u.now(), offset, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entity.NewCarFollowFileDTO, 0, len(rows))
	for _, row := range rows {
		dto := entity.ToNewCarFollowFileDTO(row)
		u.fillOwnerDisplay(ctx, &dto)
		out = append(out, dto)
	}
	return out, total, nil
}

func (u *Service) Get(ctx context.Context, actorID string, fileID int64) (*entity.NewCarFollowFileDTO, error) {
	row, err := u.loadVisibleFile(ctx, actorID, fileID)
	if err != nil {
		return nil, err
	}
	dto := entity.ToNewCarFollowFileDTO(*row)
	u.fillOwnerDisplay(ctx, &dto)
	return &dto, nil
}

func (u *Service) Create(ctx context.Context, actorID string, in CreateInput) (*entity.NewCarFollowFileDTO, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	level, ok := entity.NormalizeFollowLevel(in.FollowLevel)
	if !ok {
		return nil, ErrBadLevel
	}
	stage := entity.FollowStageNew
	if strings.TrimSpace(in.Stage) != "" {
		st, ok := entity.ParseFollowFileStage(in.Stage)
		if !ok {
			return nil, ErrBadStage
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
			return nil, ErrBadCustomer
		}
	} else {
		name := strings.TrimSpace(in.DisplayName)
		phone := strings.TrimSpace(in.Phone)
		if name == "" || phone == "" {
			return nil, ErrBadCustomer
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
		return nil, ErrBadCustomer
	}
	if entity.FollowFileIsOpen(stage) {
		existing, err := u.repo.FindOpenFileByCustomer(ctx, storeID, cust.CustomerID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return nil, ErrDuplicate
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
	u.fillOwnerDisplay(ctx, &dto)
	return &dto, nil
}

func (u *Service) Patch(
	ctx context.Context, actorID string, fileID int64, in PatchInput,
) (*entity.NewCarFollowFileDTO, error) {
	row, err := u.loadVisibleFile(ctx, actorID, fileID)
	if err != nil {
		return nil, err
	}

	syncFollow := false
	if in.FollowLevel != nil {
		lv, ok := entity.NormalizeFollowLevel(*in.FollowLevel)
		if !ok {
			return nil, ErrBadLevel
		}
		row.FollowLevel = lv
	}
	if in.Stage != nil {
		st, ok := entity.ParseFollowFileStage(*in.Stage)
		if !ok {
			return nil, ErrBadStage
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
	u.fillOwnerDisplay(ctx, &dto)
	return &dto, nil
}

func (u *Service) ListLogs(
	ctx context.Context, actorID string, fileID int64, page, size int,
) ([]entity.NewCarFollowLogDTO, int64, error) {
	if _, err := u.loadVisibleFile(ctx, actorID, fileID); err != nil {
		return nil, 0, err
	}
	_, size, offset := shared.PageOffset(page, size, 20, 50)
	rows, total, err := u.repo.ListLogs(ctx, fileID, offset, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]entity.NewCarFollowLogDTO, 0, len(rows))
	for _, row := range rows {
		dto := entity.ToNewCarFollowLogDTO(row)
		if name, _, err := u.repo.GetUserBrief(ctx, row.AuthorUserID); err == nil {
			dto.AuthorDisplayName = name
		}
		out = append(out, dto)
	}
	return out, total, nil
}

func (u *Service) CreateLog(
	ctx context.Context, actorID string, fileID int64, in CreateFollowLogInput,
) (*entity.NewCarFollowLogDTO, error) {
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, ErrBadLogBody
	}
	row, err := u.loadVisibleFile(ctx, actorID, fileID)
	if err != nil {
		return nil, err
	}

	now := u.now()
	logLevel := ""
	if strings.TrimSpace(in.FollowLevel) != "" {
		lv, ok := entity.NormalizeFollowLevel(in.FollowLevel)
		if !ok {
			return nil, ErrBadLevel
		}
		logLevel = lv
		row.FollowLevel = lv
	}
	syncCustomer := false
	if in.TouchNext {
		row.NextFollowUpAt = in.NextFollowUpAt
		syncCustomer = true
	}
	row.LastFollowAt = &now
	row.UpdatedAt = now
	if entity.FollowFileIsOpen(row.Stage) && row.Stage == entity.FollowStageNew {
		row.Stage = entity.FollowStageFollowing
	}

	log := &entity.WysNewCarFollowLog{
		FileID:         fileID,
		AuthorUserID:   actorID,
		Body:           body,
		FollowLevel:    logLevel,
		NextFollowUpAt: nil,
		CreatedAt:      now,
	}
	if in.TouchNext {
		log.NextFollowUpAt = in.NextFollowUpAt
	}

	if err := u.repo.CreateLogAndTouchFile(ctx, log, row, syncCustomer); err != nil {
		return nil, err
	}
	dto := entity.ToNewCarFollowLogDTO(*log)
	if name, _, err := u.repo.GetUserBrief(ctx, actorID); err == nil {
		dto.AuthorDisplayName = name
	}
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

// ownerScope 店管 → 空串（全店）；否则 actorID。
func (u *Service) ownerScope(ctx context.Context, actorID string, storeID int) (string, error) {
	ok, err := u.isStoreAdmin(ctx, actorID, storeID)
	if err != nil {
		return "", err
	}
	if ok {
		return "", nil
	}
	return actorID, nil
}

func (u *Service) isStoreAdmin(ctx context.Context, actorID string, storeID int) (bool, error) {
	if u.accessSvc == nil {
		return false, nil
	}
	sid := storeID
	return u.accessSvc.Can(ctx, actorID, followStoreAdminPerm, &sid)
}

func (u *Service) loadVisibleFile(ctx context.Context, actorID string, fileID int64) (*entity.WysNewCarFollowFile, error) {
	storeID, err := u.requireCurrentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	row, err := u.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if row.StoreID != storeID {
		return nil, ErrNotFound
	}
	admin, err := u.isStoreAdmin(ctx, actorID, storeID)
	if err != nil {
		return nil, err
	}
	if !admin && row.OwnerUserID != actorID {
		return nil, ErrNotFound
	}
	return row, nil
}

func (u *Service) fillOwnerDisplay(ctx context.Context, dto *entity.NewCarFollowFileDTO) {
	if dto == nil || dto.OwnerUserID == "" {
		return
	}
	if name, _, err := u.repo.GetUserBrief(ctx, dto.OwnerUserID); err == nil {
		dto.OwnerDisplayName = name
	}
}

// parseFileID 路径 id。
func parseFileID(raw string) (int64, error) {
	return shared.ParsePositiveInt64(raw, ErrNotFound)
}
