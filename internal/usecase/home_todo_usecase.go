// home_todo_usecase.go 入店申请与首页待办卡聚合。
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrJoinApplicationNotFound   = repository.ErrJoinApplicationNotFound
	ErrJoinApplicationInvalid    = repository.ErrJoinApplicationInvalid
	ErrJoinApplicationNotPending = repository.ErrJoinApplicationNotPending
	ErrJoinApplicationDuplicate  = repository.ErrJoinApplicationDuplicate
	ErrJoinAlreadyMember         = repository.ErrJoinAlreadyMember
)

// 首期：店管待办卡用 role.assign_store（store_admin 有、store_staff 无）。
const todoStoreAdminPerm = entity.PermRoleAssignStore

// HomeTodoUsecase 首页待办 + 四域读写。
type HomeTodoUsecase struct {
	repo   repository.HomeTodoRepository
	access *AccessUsecase
	now    func() time.Time
}

// NewHomeTodoUsecase 创建。
func NewHomeTodoUsecase(repo repository.HomeTodoRepository, access *AccessUsecase) *HomeTodoUsecase {
	return &HomeTodoUsecase{
		repo:   repo,
		access: access,
		now:    time.Now,
	}
}

// ListTodoCards GET 聚合：按类型优先级，省略 count=0 / 无权限。
// 若门店启用装箱演示规格，则按 大/中/小张数展开（同尺寸可多张，便于 UI 联调）。
func (u *HomeTodoUsecase) ListTodoCards(ctx context.Context, actorID string) ([]entity.HomeTodoCard, error) {
	storeID, err := u.currentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if storeID == 0 {
		return []entity.HomeTodoCard{}, nil
	}
	now := u.now()
	today := shanghaiDate(now)

	var joinN, followN, apptN, orderN int64

	okMember, err := u.can(ctx, actorID, entity.PermMemberWrite, storeID)
	if err != nil {
		return nil, err
	}
	if okMember {
		joinN, err = u.repo.CountPendingJoinApplications(ctx, storeID)
		if err != nil {
			return nil, err
		}
	}

	okAdmin, err := u.can(ctx, actorID, todoStoreAdminPerm, storeID)
	if err != nil {
		return nil, err
	}
	if okAdmin {
		followN, err = u.repo.CountOverdueCustomers(ctx, storeID, now)
		if err != nil {
			return nil, err
		}
		apptN, err = u.repo.CountPendingAppointments(ctx, storeID, today)
		if err != nil {
			return nil, err
		}
		orderN, err = u.repo.CountPendingReviewOrders(ctx, storeID)
		if err != nil {
			return nil, err
		}
	}

	spec, err := u.repo.GetPackingDemoSpec(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if spec != nil && (spec.LargeN > 0 || spec.MediumN > 0 || spec.SmallN > 0) {
		return buildPackingDemoCards(spec, okMember, okAdmin, joinN, followN, apptN, orderN), nil
	}

	out := make([]entity.HomeTodoCard, 0, 4)
	if okMember && joinN > 0 {
		out = append(out, entity.HomeTodoCard{
			Type:        entity.TodoTypePartnerPending,
			Title:       "新伙伴待确认",
			Subtitle:    fmt.Sprintf("%d 位新成员等待审核", joinN),
			ActionLabel: "去处理",
			ActionRoute: "/home/todo/partner-pending",
			Count:       joinN,
		})
	}
	if okAdmin {
		if followN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeFollowUpCustomer,
				Title:       "待跟进客户",
				Subtitle:    fmt.Sprintf("%d 位客户待跟进", followN),
				ActionLabel: "去查看",
				ActionRoute: "/home/todo/follow-up-customers",
				Count:       followN,
			})
		}
		if apptN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeAfterSalesAppointment,
				Title:       "售后预约",
				Subtitle:    fmt.Sprintf("%d 条待处理预约", apptN),
				ActionLabel: "去查看",
				ActionRoute: "/home/todo/after-sales-appointments",
				Count:       apptN,
			})
		}
		if orderN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeOrderPendingReview,
				Title:       "订单待审核",
				Subtitle:    fmt.Sprintf("%d 笔订单待审核", orderN),
				ActionLabel: "去处理",
				ActionRoute: "/home/todo/order-pending-review",
				Count:       orderN,
			})
		}
	}
	return out, nil
}

func buildPackingDemoCards(
	spec *repository.HomeTodoPackingDemoSpec,
	okMember, okAdmin bool,
	joinN, followN, apptN, orderN int64,
) []entity.HomeTodoCard {
	out := make([]entity.HomeTodoCard, 0, spec.LargeN+spec.MediumN+spec.SmallN)

	for i := 0; i < spec.LargeN; i++ {
		if !okMember || joinN <= 0 {
			break
		}
		out = append(out, entity.HomeTodoCard{
			Type:        entity.TodoTypePartnerPending,
			Title:       "新伙伴待确认",
			Subtitle:    fmt.Sprintf("%d 位新成员等待审核", joinN),
			ActionLabel: "去处理",
			ActionRoute: "/home/todo/partner-pending",
			Count:       joinN,
		})
	}

	mediums := []entity.HomeTodoCard{
		{
			Type:        entity.TodoTypeFollowUpCustomer,
			Title:       "待跟进客户",
			Subtitle:    fmt.Sprintf("%d 位客户待跟进", followN),
			ActionLabel: "去查看",
			ActionRoute: "/home/todo/follow-up-customers",
			Count:       followN,
		},
		{
			Type:        entity.TodoTypeAfterSalesAppointment,
			Title:       "售后预约",
			Subtitle:    fmt.Sprintf("%d 条待处理预约", apptN),
			ActionLabel: "去查看",
			ActionRoute: "/home/todo/after-sales-appointments",
			Count:       apptN,
		},
		{
			Type:        entity.TodoTypeFollowUpCustomer,
			Title:       "高意向回访",
			Subtitle:    fmt.Sprintf("%d 位需今日回访", followN),
			ActionLabel: "去查看",
			ActionRoute: "/home/todo/follow-up-customers",
			Count:       followN,
		},
	}
	for i := 0; i < spec.MediumN && i < len(mediums); i++ {
		if !okAdmin {
			break
		}
		c := mediums[i]
		if c.Count <= 0 {
			c.Count = 1
			c.Subtitle = c.Title
		}
		out = append(out, c)
	}

	for i := 0; i < spec.SmallN; i++ {
		if !okAdmin || orderN <= 0 {
			break
		}
		out = append(out, entity.HomeTodoCard{
			Type:        entity.TodoTypeOrderPendingReview,
			Title:       fmt.Sprintf("订单待审核·%d", i+1),
			Subtitle:    fmt.Sprintf("共 %d 笔待审", orderN),
			ActionLabel: "去处理",
			ActionRoute: "/home/todo/order-pending-review",
			Count:       1,
		})
	}
	return out
}

// ApplyToStore 提交入店申请。
func (u *HomeTodoUsecase) ApplyToStore(ctx context.Context, applicantID string, storeID int) (*entity.WysStoreJoinApplication, error) {
	if applicantID == "" || storeID <= 0 {
		return nil, ErrJoinApplicationInvalid
	}
	if u.access != nil {
		if _, err := u.access.repo.GetMember(ctx, applicantID, storeID); err == nil {
			return nil, ErrJoinAlreadyMember
		}
	}
	app := &entity.WysStoreJoinApplication{
		StoreID:         storeID,
		ApplicantUserID: applicantID,
		Status:          entity.JoinStatusPending,
	}
	if err := u.repo.CreateJoinApplication(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

// ListPendingJoinApplications 当前店 pending 列表。
func (u *HomeTodoUsecase) ListPendingJoinApplications(ctx context.Context, actorID string) ([]entity.WysStoreJoinApplication, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, entity.PermMemberWrite)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingJoinApplications(ctx, storeID)
}

// ApproveJoinApplication 确认入店。
func (u *HomeTodoUsecase) ApproveJoinApplication(ctx context.Context, actorID string, applicationID int64) error {
	app, err := u.repo.GetJoinApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	if app.Status != entity.JoinStatusPending {
		return ErrJoinApplicationNotPending
	}
	if err := u.access.RequirePermission(ctx, actorID, entity.PermMemberWrite, &app.StoreID); err != nil {
		return err
	}
	if _, err := u.access.UpsertMember(ctx, actorID, app.StoreID, app.ApplicantUserID, 0); err != nil {
		return err
	}
	return u.repo.UpdateJoinApplicationStatus(ctx, applicationID, entity.JoinStatusApproved, actorID)
}

// RejectJoinApplication 拒绝入店。
func (u *HomeTodoUsecase) RejectJoinApplication(ctx context.Context, actorID string, applicationID int64) error {
	app, err := u.repo.GetJoinApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	if app.Status != entity.JoinStatusPending {
		return ErrJoinApplicationNotPending
	}
	if err := u.access.RequirePermission(ctx, actorID, entity.PermMemberWrite, &app.StoreID); err != nil {
		return err
	}
	return u.repo.UpdateJoinApplicationStatus(ctx, applicationID, entity.JoinStatusRejected, actorID)
}

// ListOverdueCustomers 待跟进客户。
func (u *HomeTodoUsecase) ListOverdueCustomers(ctx context.Context, actorID string) ([]entity.WysStoreCustomer, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, todoStoreAdminPerm)
	if err != nil {
		return nil, err
	}
	return u.repo.ListOverdueCustomers(ctx, storeID, u.now())
}

// ListPendingAppointments 售后预约。
func (u *HomeTodoUsecase) ListPendingAppointments(ctx context.Context, actorID string) ([]entity.WysAfterSalesAppointment, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, todoStoreAdminPerm)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingAppointments(ctx, storeID, shanghaiDate(u.now()))
}

// ListPendingReviewOrders 店务审核单。
func (u *HomeTodoUsecase) ListPendingReviewOrders(ctx context.Context, actorID string) ([]entity.WysStoreReviewOrder, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, todoStoreAdminPerm)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingReviewOrders(ctx, storeID)
}

// EnsureSeed LAN 种子。
func (u *HomeTodoUsecase) EnsureSeed(ctx context.Context, storeID int, applicantUserID string) error {
	if storeID <= 0 {
		return nil
	}
	return u.repo.EnsureSeed(ctx, storeID, applicantUserID)
}

func (u *HomeTodoUsecase) currentStore(ctx context.Context, actorID string) (int, error) {
	if u.access == nil {
		return 0, nil
	}
	cur, err := u.access.repo.CurrentStoreID(ctx, actorID)
	if err != nil || cur == nil {
		return 0, err
	}
	return *cur, nil
}

func (u *HomeTodoUsecase) can(ctx context.Context, actorID, perm string, storeID int) (bool, error) {
	if u.access == nil {
		return false, nil
	}
	err := u.access.RequirePermission(ctx, actorID, perm, &storeID)
	if err == nil {
		return true, nil
	}
	if err == ErrAccessForbidden {
		return false, nil
	}
	return false, err
}

func (u *HomeTodoUsecase) requireStorePerm(ctx context.Context, actorID, perm string) (int, error) {
	storeID, err := u.currentStore(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if storeID == 0 {
		return 0, ErrAccessForbidden
	}
	if err := u.access.RequirePermission(ctx, actorID, perm, &storeID); err != nil {
		return 0, err
	}
	return storeID, nil
}

func shanghaiDate(t time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	local := t.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}
