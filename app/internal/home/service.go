// home_todo_usecase.go 入店申请与首页待办卡聚合。
package home

import (
	"context"
	"errors"
	"fmt"
	"time"

	"app/internal/access"
	"model/entity"
)

// 首期：店管待办卡用 role.assign_store（store_admin 有、store_staff 无）。
const todoStoreAdminPerm = entity.PermRoleAssignStore

// Service 首页待办 + 四域读写。
type Service struct {
	repo      Repository
	accessSvc *access.Service
	now       func() time.Time
}

// NewService 创建。
func NewService(repo Repository, accessSvc *access.Service) *Service {
	return &Service{
		repo:      repo,
		accessSvc: accessSvc,
		now:       time.Now,
	}
}

// ListTodoCards GET 聚合：按类型优先级，省略 count=0 / 无权限。
// 若门店启用装箱演示规格，则按 大/中/小张数展开（同尺寸可多张，便于 UI 联调）。
func (u *Service) ListTodoCards(ctx context.Context, actorID string) ([]entity.HomeTodoCard, error) {
	storeID, err := u.currentStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if storeID == 0 {
		return []entity.HomeTodoCard{}, nil
	}
	now := u.now()
	today := shanghaiDate(now)

	var joinN, followN, apptN, orderN, usedCarN int64

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
		usedCarN, err = u.repo.CountPendingUsedCarOrders(ctx, storeID)
		if err != nil {
			return nil, err
		}
	}

	spec, err := u.repo.GetPackingDemoSpec(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if spec != nil && (spec.LargeN > 0 || spec.MediumN > 0 || spec.SmallN > 0) {
		return buildPackingDemoCards(spec, okMember, okAdmin, joinN, followN, apptN, orderN, usedCarN), nil
	}

	out := make([]entity.HomeTodoCard, 0, 5)
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
		if usedCarN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeUsedCarPendingReview,
				Title:       "二手车待审",
				Subtitle:    fmt.Sprintf("%d 笔二手车业务单待审核", usedCarN),
				ActionLabel: "去查看",
				ActionRoute: "/home/used_car",
				Count:       usedCarN,
			})
		}
	}
	return out, nil
}

func buildPackingDemoCards(
	spec *HomeTodoPackingDemoSpec,
	okMember, okAdmin bool,
	joinN, followN, apptN, orderN, usedCarN int64,
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

	smallSlots := 0
	for i := 0; i < spec.SmallN; i++ {
		if !okAdmin {
			break
		}
		if smallSlots%2 == 0 && orderN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeOrderPendingReview,
				Title:       fmt.Sprintf("订单待审核·%d", i+1),
				Subtitle:    fmt.Sprintf("共 %d 笔待审", orderN),
				ActionLabel: "去处理",
				ActionRoute: "/home/todo/order-pending-review",
				Count:       1,
			})
			smallSlots++
			continue
		}
		if usedCarN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeUsedCarPendingReview,
				Title:       fmt.Sprintf("二手车待审·%d", i+1),
				Subtitle:    fmt.Sprintf("共 %d 笔待审", usedCarN),
				ActionLabel: "去查看",
				ActionRoute: "/home/used_car",
				Count:       1,
			})
			smallSlots++
			continue
		}
		if orderN > 0 {
			out = append(out, entity.HomeTodoCard{
				Type:        entity.TodoTypeOrderPendingReview,
				Title:       fmt.Sprintf("订单待审核·%d", i+1),
				Subtitle:    fmt.Sprintf("共 %d 笔待审", orderN),
				ActionLabel: "去处理",
				ActionRoute: "/home/todo/order-pending-review",
				Count:       1,
			})
			smallSlots++
		}
	}
	return out
}

// ApplyToStore 提交入店申请。
func (u *Service) ApplyToStore(ctx context.Context, applicantID string, storeID int) (*entity.WysStoreJoinApplication, error) {
	if applicantID == "" || storeID <= 0 {
		return nil, ErrJoinApplicationInvalid
	}
	if u.accessSvc != nil {
		if isMember, err := u.accessSvc.IsMember(ctx, applicantID, storeID); err == nil && isMember {
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
func (u *Service) ListPendingJoinApplications(ctx context.Context, actorID string) ([]entity.WysStoreJoinApplication, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, entity.PermMemberWrite)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingJoinApplications(ctx, storeID)
}

// ApproveJoinApplication 确认入店。
func (u *Service) ApproveJoinApplication(ctx context.Context, actorID string, applicationID int64) error {
	app, err := u.repo.GetJoinApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	if app.Status != entity.JoinStatusPending {
		return ErrJoinApplicationNotPending
	}
	if err := u.accessSvc.RequirePermission(ctx, actorID, entity.PermMemberWrite, &app.StoreID); err != nil {
		return err
	}
	if _, err := u.accessSvc.UpsertMember(ctx, actorID, app.StoreID, app.ApplicantUserID, 0); err != nil {
		return err
	}
	return u.repo.UpdateJoinApplicationStatus(ctx, applicationID, entity.JoinStatusApproved, actorID)
}

// RejectJoinApplication 拒绝入店。
func (u *Service) RejectJoinApplication(ctx context.Context, actorID string, applicationID int64) error {
	app, err := u.repo.GetJoinApplication(ctx, applicationID)
	if err != nil {
		return err
	}
	if app.Status != entity.JoinStatusPending {
		return ErrJoinApplicationNotPending
	}
	if err := u.accessSvc.RequirePermission(ctx, actorID, entity.PermMemberWrite, &app.StoreID); err != nil {
		return err
	}
	return u.repo.UpdateJoinApplicationStatus(ctx, applicationID, entity.JoinStatusRejected, actorID)
}

// ListOverdueCustomers 待跟进客户。
func (u *Service) ListOverdueCustomers(ctx context.Context, actorID string) ([]entity.WysStoreCustomer, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, todoStoreAdminPerm)
	if err != nil {
		return nil, err
	}
	return u.repo.ListOverdueCustomers(ctx, storeID, u.now())
}

// ListPendingAppointments 售后预约。
func (u *Service) ListPendingAppointments(ctx context.Context, actorID string) ([]entity.WysAfterSalesAppointment, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, todoStoreAdminPerm)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingAppointments(ctx, storeID, shanghaiDate(u.now()))
}

// ListPendingReviewOrders 店务审核单。
func (u *Service) ListPendingReviewOrders(ctx context.Context, actorID string) ([]entity.WysStoreReviewOrder, error) {
	storeID, err := u.requireStorePerm(ctx, actorID, todoStoreAdminPerm)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingReviewOrders(ctx, storeID)
}

// EnsureSeed LAN 种子。
func (u *Service) EnsureSeed(ctx context.Context, storeID int, applicantUserID string) error {
	if storeID <= 0 {
		return nil
	}
	return u.repo.EnsureSeed(ctx, storeID, applicantUserID)
}

func (u *Service) currentStore(ctx context.Context, actorID string) (int, error) {
	if u.accessSvc == nil {
		return 0, nil
	}
	cur, err := u.accessSvc.CurrentStoreID(ctx, actorID)
	if err != nil || cur == nil {
		return 0, err
	}
	return *cur, nil
}

func (u *Service) can(ctx context.Context, actorID, perm string, storeID int) (bool, error) {
	if u.accessSvc == nil {
		return false, nil
	}
	err := u.accessSvc.RequirePermission(ctx, actorID, perm, &storeID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, access.ErrForbidden) {
		return false, nil
	}
	return false, err
}

func (u *Service) requireStorePerm(ctx context.Context, actorID, perm string) (int, error) {
	storeID, err := u.currentStore(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if storeID == 0 {
		return 0, access.ErrForbidden
	}
	if err := u.accessSvc.RequirePermission(ctx, actorID, perm, &storeID); err != nil {
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
