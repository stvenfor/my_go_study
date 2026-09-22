package entity

import "time"

const (
	WysPointsWalletTable  = "wys_points_wallet"
	WysPointsLedgerTable  = "wys_points_ledger"
	WysCheckInTable       = "wys_check_ins"
	WysTaskClaimTable     = "wys_task_claims"

	PointsReasonCheckIn     = "check_in"
	PointsReasonTaskLogin   = "task_login"
	PointsReasonTaskPost    = "task_post"
	PointsReasonTaskOrder   = "task_order"
	PointsReasonMallPay     = "mall_pay"
	PointsReasonMallRefund  = "mall_refund"

	TaskLogin = "login"
	TaskPost  = "post"
	TaskOrder = "order"

	CheckInRewardBase   int64 = 5
	CheckInRewardDay3   int64 = 10
	CheckInRewardDay7   int64 = 20
	TaskRewardLogin     int64 = 5
	TaskRewardPost      int64 = 10
	TaskRewardOrder     int64 = 20
)

// WysPointsWallet 账号全局积分余额。
type WysPointsWallet struct {
	UserID    string    `json:"user_id" gorm:"primaryKey"`
	Balance   int64     `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysPointsWallet) TableName() string { return WysPointsWalletTable }

// WysPointsLedger 积分流水。
type WysPointsLedger struct {
	LedgerID  int64     `json:"ledger_id" gorm:"primaryKey"`
	UserID    string    `json:"user_id"`
	Delta     int64     `json:"delta"`
	Balance   int64     `json:"balance"`
	Reason    string    `json:"reason"`
	RefID     string    `json:"ref_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (WysPointsLedger) TableName() string { return WysPointsLedgerTable }

// WysCheckIn 某用户在某一上海自然日的签到记录。
type WysCheckIn struct {
	UserID    string    `json:"user_id" gorm:"primaryKey"`
	Day       string    `json:"day" gorm:"primaryKey"` // yyyy-MM-dd Asia/Shanghai
	Streak    int       `json:"streak"`
	Points    int64     `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

func (WysCheckIn) TableName() string { return WysCheckInTable }

// WysTaskClaim 某用户某日某任务的领取记录。
type WysTaskClaim struct {
	UserID    string    `json:"user_id" gorm:"primaryKey"`
	Day       string    `json:"day" gorm:"primaryKey"`
	TaskCode  string    `json:"task_code" gorm:"primaryKey"`
	Points    int64     `json:"points"`
	CreatedAt time.Time `json:"created_at"`
}

func (WysTaskClaim) TableName() string { return WysTaskClaimTable }

// CheckInRewardForStreak 连签第 n 日（含当日）应得积分；7 日一轮。
func CheckInRewardForStreak(streak int) int64 {
	if streak < 1 {
		return CheckInRewardBase
	}
	pos := ((streak - 1) % 7) + 1
	switch pos {
	case 3:
		return CheckInRewardDay3
	case 7:
		return CheckInRewardDay7
	default:
		return CheckInRewardBase
	}
}

// PointsStatus 签到页/弹窗读模型。
type PointsStatus struct {
	Balance         int64            `json:"balance"`
	CheckedInToday  bool             `json:"checked_in_today"`
	Streak          int              `json:"streak"`
	TodayReward     int64            `json:"today_reward"`
	Calendar        []CheckInDayView `json:"calendar"`
}

// CheckInDayView 日历一日。
type CheckInDayView struct {
	Day     string `json:"day"`
	Signed  bool   `json:"signed"`
	Reward  int64  `json:"reward"`
	IsToday bool   `json:"is_today"`
}

// GrowthTaskView 成长任务读模型。
type GrowthTaskView struct {
	Code      string `json:"code"`
	Title     string `json:"title"`
	Reward    int64  `json:"reward"`
	Progress  string `json:"progress"` // incomplete | claimable | claimed
}

// CheckInResult 签到成功结果。
type CheckInResult struct {
	Points  int64 `json:"points"`
	Balance int64 `json:"balance"`
	Streak  int   `json:"streak"`
	Day     string `json:"day"`
}
