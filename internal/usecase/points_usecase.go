package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrPointsAlreadyCheckedIn = repository.ErrPointsAlreadyCheckedIn
	ErrPointsInsufficient     = repository.ErrPointsInsufficient
	ErrPointsTaskNotClaimable = repository.ErrPointsTaskNotClaimable
	ErrPointsTaskUnknown      = repository.ErrPointsTaskUnknown
)

// TaskProgressSource 成长任务完成事实（社区/商城/会话）。
type TaskProgressSource interface {
	HasLoggedInToday(ctx context.Context, userID, day string) (bool, error)
	HasPostedToday(ctx context.Context, userID, day string) (bool, error)
	HasPaidOrderToday(ctx context.Context, userID, day string) (bool, error)
}

// PointsUsecase 积分账本、签到、成长任务。
type PointsUsecase struct {
	repo   repository.PointsRepository
	tasks  TaskProgressSource
	nowFn  func() time.Time
	loc    *time.Location
}

func NewPointsUsecase(repo repository.PointsRepository, tasks TaskProgressSource) *PointsUsecase {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return &PointsUsecase{
		repo:  repo,
		tasks: tasks,
		nowFn: time.Now,
		loc:   loc,
	}
}

// WithClock 测试用固定时钟。
func (u *PointsUsecase) WithClock(now func() time.Time) *PointsUsecase {
	u.nowFn = now
	return u
}

func (u *PointsUsecase) shanghaiDay(t time.Time) string {
	return t.In(u.loc).Format("2006-01-02")
}

func (u *PointsUsecase) today() string {
	return u.shanghaiDay(u.nowFn())
}

// GetStatus 余额 + 今日签到态 + 近 7 日日历。
func (u *PointsUsecase) GetStatus(ctx context.Context, userID string) (*entity.PointsStatus, error) {
	bal, err := u.repo.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	today := u.today()
	ci, err := u.repo.GetCheckIn(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	streak := 0
	if ci != nil {
		streak = ci.Streak
	} else {
		streak = u.previewStreak(ctx, userID, today)
	}
	from := u.nowFn().In(u.loc).AddDate(0, 0, -6).Format("2006-01-02")
	rows, err := u.repo.ListCheckIns(ctx, userID, from, today)
	if err != nil {
		return nil, err
	}
	signed := map[string]entity.WysCheckIn{}
	for _, r := range rows {
		signed[r.Day] = r
	}
	cal := make([]entity.CheckInDayView, 0, 7)
	for i := 6; i >= 0; i-- {
		d := u.nowFn().In(u.loc).AddDate(0, 0, -i).Format("2006-01-02")
		view := entity.CheckInDayView{Day: d, IsToday: d == today}
		if row, ok := signed[d]; ok {
			view.Signed = true
			view.Reward = row.Points
		} else if d == today {
			view.Reward = entity.CheckInRewardForStreak(streak)
		} else {
			view.Reward = entity.CheckInRewardBase
		}
		cal = append(cal, view)
	}
	reward := entity.CheckInRewardForStreak(streak)
	if ci != nil {
		reward = ci.Points
	}
	return &entity.PointsStatus{
		Balance:        bal,
		CheckedInToday: ci != nil,
		Streak:         streak,
		TodayReward:    reward,
		Calendar:       cal,
	}, nil
}

func (u *PointsUsecase) previewStreak(ctx context.Context, userID, today string) int {
	yday := u.nowFn().In(u.loc).AddDate(0, 0, -1).Format("2006-01-02")
	prev, err := u.repo.GetCheckIn(ctx, userID, yday)
	if err != nil || prev == nil {
		return 1
	}
	return prev.Streak + 1
}

// CheckIn 完成当日签到并入账。
func (u *PointsUsecase) CheckIn(ctx context.Context, userID string) (*entity.CheckInResult, error) {
	today := u.today()
	existing, err := u.repo.GetCheckIn(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPointsAlreadyCheckedIn
	}
	streak := u.previewStreak(ctx, userID, today)
	pts := entity.CheckInRewardForStreak(streak)
	row := &entity.WysCheckIn{
		UserID:    userID,
		Day:       today,
		Streak:    streak,
		Points:    pts,
		CreatedAt: u.nowFn().In(u.loc),
	}
	bal, err := u.repo.CheckInAtomic(ctx, row)
	if err != nil {
		if errors.Is(err, repository.ErrPointsAlreadyCheckedIn) {
			return nil, ErrPointsAlreadyCheckedIn
		}
		return nil, err
	}
	return &entity.CheckInResult{
		Points:  pts,
		Balance: bal,
		Streak:  streak,
		Day:     today,
	}, nil
}

// Credit / Debit 供商城支付调用。
func (u *PointsUsecase) Credit(ctx context.Context, userID string, delta int64, reason, refID string) (int64, error) {
	return u.repo.Credit(ctx, userID, delta, reason, refID)
}

func (u *PointsUsecase) Debit(ctx context.Context, userID string, delta int64, reason, refID string) (int64, error) {
	return u.repo.Debit(ctx, userID, delta, reason, refID)
}

func (u *PointsUsecase) GetBalance(ctx context.Context, userID string) (int64, error) {
	return u.repo.GetBalance(ctx, userID)
}

// ListTasks 成长任务进度。
func (u *PointsUsecase) ListTasks(ctx context.Context, userID string) ([]entity.GrowthTaskView, error) {
	today := u.today()
	defs := []struct {
		code, title string
		reward      int64
		done        func(context.Context, string, string) (bool, error)
	}{
		{entity.TaskLogin, "每日登录", entity.TaskRewardLogin, u.taskLoggedIn},
		{entity.TaskPost, "发一条动态", entity.TaskRewardPost, u.taskPosted},
		{entity.TaskOrder, "商城下单", entity.TaskRewardOrder, u.taskPaidOrder},
	}
	out := make([]entity.GrowthTaskView, 0, len(defs))
	for _, d := range defs {
		claim, err := u.repo.GetTaskClaim(ctx, userID, today, d.code)
		if err != nil {
			return nil, err
		}
		v := entity.GrowthTaskView{Code: d.code, Title: d.title, Reward: d.reward, Progress: "incomplete"}
		if claim != nil {
			v.Progress = "claimed"
			out = append(out, v)
			continue
		}
		ok, err := d.done(ctx, userID, today)
		if err != nil {
			return nil, err
		}
		if ok {
			v.Progress = "claimable"
		}
		out = append(out, v)
	}
	return out, nil
}

func (u *PointsUsecase) taskLoggedIn(ctx context.Context, userID, day string) (bool, error) {
	if u.tasks == nil {
		return true, nil // 已鉴权进 API 即视为当日已登录
	}
	return u.tasks.HasLoggedInToday(ctx, userID, day)
}

func (u *PointsUsecase) taskPosted(ctx context.Context, userID, day string) (bool, error) {
	if u.tasks == nil {
		return false, nil
	}
	return u.tasks.HasPostedToday(ctx, userID, day)
}

func (u *PointsUsecase) taskPaidOrder(ctx context.Context, userID, day string) (bool, error) {
	if u.tasks == nil {
		return false, nil
	}
	return u.tasks.HasPaidOrderToday(ctx, userID, day)
}

// ClaimTask 领取成长任务奖励。
func (u *PointsUsecase) ClaimTask(ctx context.Context, userID, taskCode string) (*entity.CheckInResult, error) {
	var reward int64
	var reason string
	switch taskCode {
	case entity.TaskLogin:
		reward, reason = entity.TaskRewardLogin, entity.PointsReasonTaskLogin
	case entity.TaskPost:
		reward, reason = entity.TaskRewardPost, entity.PointsReasonTaskPost
	case entity.TaskOrder:
		reward, reason = entity.TaskRewardOrder, entity.PointsReasonTaskOrder
	default:
		return nil, ErrPointsTaskUnknown
	}
	today := u.today()
	existing, err := u.repo.GetTaskClaim(ctx, userID, today, taskCode)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		bal, err := u.repo.GetBalance(ctx, userID)
		if err != nil {
			return nil, err
		}
		return &entity.CheckInResult{Points: 0, Balance: bal, Day: today}, nil
	}
	ok := false
	switch taskCode {
	case entity.TaskLogin:
		ok, err = u.taskLoggedIn(ctx, userID, today)
	case entity.TaskPost:
		ok, err = u.taskPosted(ctx, userID, today)
	case entity.TaskOrder:
		ok, err = u.taskPaidOrder(ctx, userID, today)
	}
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPointsTaskNotClaimable
	}
	claim := &entity.WysTaskClaim{
		UserID: userID, Day: today, TaskCode: taskCode, Points: reward, CreatedAt: u.nowFn().In(u.loc),
	}
	bal, err := u.repo.ClaimTaskAtomic(ctx, claim, reason)
	if err != nil {
		if errors.Is(err, repository.ErrPointsAlreadyCheckedIn) {
			b, e2 := u.repo.GetBalance(ctx, userID)
			if e2 != nil {
				return nil, e2
			}
			return &entity.CheckInResult{Points: 0, Balance: b, Day: today}, nil
		}
		return nil, err
	}
	return &entity.CheckInResult{Points: reward, Balance: bal, Day: today}, nil
}
