package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type memPointsRepo struct {
	mu      sync.Mutex
	balance map[string]int64
	checkIn map[string]entity.WysCheckIn // userID|day
	claims  map[string]entity.WysTaskClaim
}

func newMemPointsRepo() *memPointsRepo {
	return &memPointsRepo{
		balance: map[string]int64{},
		checkIn: map[string]entity.WysCheckIn{},
		claims:  map[string]entity.WysTaskClaim{},
	}
}

func (m *memPointsRepo) key(userID, day string) string { return userID + "|" + day }

func (m *memPointsRepo) GetBalance(_ context.Context, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balance[userID], nil
}

func (m *memPointsRepo) Credit(_ context.Context, userID string, delta int64, _, _ string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balance[userID] += delta
	return m.balance[userID], nil
}

func (m *memPointsRepo) Debit(_ context.Context, userID string, delta int64, _, _ string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.balance[userID] < delta {
		return 0, repository.ErrPointsInsufficient
	}
	m.balance[userID] -= delta
	return m.balance[userID], nil
}

func (m *memPointsRepo) GetCheckIn(_ context.Context, userID, day string) (*entity.WysCheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.checkIn[m.key(userID, day)]
	if !ok {
		return nil, nil
	}
	cp := row
	return &cp, nil
}

func (m *memPointsRepo) ListCheckIns(_ context.Context, userID, fromDay, toDay string) ([]entity.WysCheckIn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []entity.WysCheckIn
	for k, v := range m.checkIn {
		if len(k) < len(userID)+1 || k[:len(userID)] != userID {
			continue
		}
		if v.Day >= fromDay && v.Day <= toDay {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *memPointsRepo) InsertCheckIn(_ context.Context, row *entity.WysCheckIn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := m.key(row.UserID, row.Day)
	if _, ok := m.checkIn[k]; ok {
		return repository.ErrPointsAlreadyCheckedIn
	}
	m.checkIn[k] = *row
	return nil
}

func (m *memPointsRepo) GetTaskClaim(_ context.Context, userID, day, taskCode string) (*entity.WysTaskClaim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.claims[userID+"|"+day+"|"+taskCode]
	if !ok {
		return nil, nil
	}
	cp := row
	return &cp, nil
}

func (m *memPointsRepo) InsertTaskClaim(_ context.Context, row *entity.WysTaskClaim) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := row.UserID + "|" + row.Day + "|" + row.TaskCode
	if _, ok := m.claims[k]; ok {
		return repository.ErrPointsAlreadyCheckedIn
	}
	m.claims[k] = *row
	return nil
}

func (m *memPointsRepo) CheckInAtomic(ctx context.Context, row *entity.WysCheckIn) (int64, error) {
	if err := m.InsertCheckIn(ctx, row); err != nil {
		return 0, err
	}
	return m.Credit(ctx, row.UserID, row.Points, entity.PointsReasonCheckIn, row.Day)
}

func (m *memPointsRepo) ClaimTaskAtomic(ctx context.Context, claim *entity.WysTaskClaim, reason string) (int64, error) {
	if err := m.InsertTaskClaim(ctx, claim); err != nil {
		return 0, err
	}
	return m.Credit(ctx, claim.UserID, claim.Points, reason, claim.Day+":"+claim.TaskCode)
}

type stubTasks struct {
	login, post, order bool
}

func (s stubTasks) HasLoggedInToday(context.Context, string, string) (bool, error) {
	return s.login, nil
}
func (s stubTasks) HasPostedToday(context.Context, string, string) (bool, error) {
	return s.post, nil
}
func (s stubTasks) HasPaidOrderToday(context.Context, string, string) (bool, error) {
	return s.order, nil
}

func shanghai(y int, m time.Month, d, h int) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Date(y, m, d, h, 0, 0, 0, loc)
}

func TestCheckIn_rewardsAndIdempotent(t *testing.T) {
	repo := newMemPointsRepo()
	uc := usecase.NewPointsUsecase(repo, stubTasks{login: true})
	day1 := shanghai(2026, 3, 1, 10)
	uc.WithClock(func() time.Time { return day1 })

	r1, err := uc.CheckIn(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if r1.Points != 5 || r1.Streak != 1 || r1.Balance != 5 {
		t.Fatalf("day1 got %+v", r1)
	}
	if _, err := uc.CheckIn(context.Background(), "u1"); err != usecase.ErrPointsAlreadyCheckedIn {
		t.Fatalf("want already checked in, got %v", err)
	}

	uc.WithClock(func() time.Time { return shanghai(2026, 3, 2, 10) })
	r2, err := uc.CheckIn(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if r2.Points != 5 || r2.Streak != 2 {
		t.Fatalf("day2 got %+v", r2)
	}

	uc.WithClock(func() time.Time { return shanghai(2026, 3, 3, 10) })
	r3, err := uc.CheckIn(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if r3.Points != 10 || r3.Streak != 3 || r3.Balance != 20 {
		t.Fatalf("day3 got %+v", r3)
	}

	for i := 4; i <= 6; i++ {
		day := i
		uc.WithClock(func() time.Time { return shanghai(2026, 3, day, 10) })
		if _, err := uc.CheckIn(context.Background(), "u1"); err != nil {
			t.Fatal(err)
		}
	}
	uc.WithClock(func() time.Time { return shanghai(2026, 3, 7, 10) })
	r7, err := uc.CheckIn(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if r7.Points != 20 || r7.Streak != 7 {
		t.Fatalf("day7 got %+v", r7)
	}
}

func TestCheckInRewardForStreak(t *testing.T) {
	cases := []struct {
		streak int
		want   int64
	}{
		{1, 5}, {2, 5}, {3, 10}, {4, 5}, {7, 20}, {8, 5}, {10, 10}, {14, 20},
	}
	for _, c := range cases {
		if got := entity.CheckInRewardForStreak(c.streak); got != c.want {
			t.Fatalf("streak %d: got %d want %d", c.streak, got, c.want)
		}
	}
}

func TestClaimTask_idempotent(t *testing.T) {
	repo := newMemPointsRepo()
	uc := usecase.NewPointsUsecase(repo, stubTasks{login: true, post: true}).
		WithClock(func() time.Time { return shanghai(2026, 3, 1, 12) })

	r, err := uc.ClaimTask(context.Background(), "u1", entity.TaskLogin)
	if err != nil {
		t.Fatal(err)
	}
	if r.Points != 5 || r.Balance != 5 {
		t.Fatalf("got %+v", r)
	}
	r2, err := uc.ClaimTask(context.Background(), "u1", entity.TaskLogin)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Points != 0 || r2.Balance != 5 {
		t.Fatalf("idempotent got %+v", r2)
	}
	if _, err := uc.ClaimTask(context.Background(), "u1", entity.TaskOrder); err != usecase.ErrPointsTaskNotClaimable {
		t.Fatalf("want not claimable, got %v", err)
	}
}

func TestDebitInsufficient(t *testing.T) {
	repo := newMemPointsRepo()
	uc := usecase.NewPointsUsecase(repo, nil)
	if _, err := uc.Debit(context.Background(), "u1", 1, entity.PointsReasonMallPay, "o1"); err != usecase.ErrPointsInsufficient {
		t.Fatalf("got %v", err)
	}
}
