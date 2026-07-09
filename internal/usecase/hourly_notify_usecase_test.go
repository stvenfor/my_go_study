package usecase_test

import (
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestHourlyNotifyUsecaseBuildPushInput(t *testing.T) {
	cfg := config.Config{
		Scheduler: config.SchedulerConfig{
			Timezone: "Asia/Shanghai",
			HourlyNotify: config.HourlyNotifyConfig{
				TitleTemplate:   "整点提醒",
				BodyTemplate:    "现在是 {{hour}}:00，{{message}}",
				DefaultMessage:  "测试消息",
				ExpiresMinutes:  120,
				Action: config.HourlyNotifyActionConfig{
					Type:  "deeplink",
					Route: "/home",
				},
			},
		},
	}
	uc := usecase.NewHourlyNotifyUsecase(cfg)
	slot := time.Date(2026, 7, 10, 11, 0, 0, 0, cfg.Scheduler.Location())

	out := uc.BuildPushInput("user-1", slot)
	if out.UserID != "user-1" || out.Title == "" || out.Body == "" {
		t.Fatalf("unexpected input: %+v", out)
	}
	if out.Extra["campaignId"] == nil || out.Extra["scheduleSlot"] == nil {
		t.Fatalf("missing extra fields: %+v", out.Extra)
	}
	if out.Extra["category"] != "scheduled" {
		t.Fatalf("category=%v", out.Extra["category"])
	}
}

func TestHourlyNotifyUsecaseMorningTemplate(t *testing.T) {
	cfg := config.Config{Scheduler: config.SchedulerConfig{Timezone: "Asia/Shanghai"}}
	uc := usecase.NewHourlyNotifyUsecase(cfg)
	slot := time.Date(2026, 7, 10, 10, 0, 0, 0, cfg.Scheduler.Location())

	out := uc.BuildPushInput("u1", slot)
	if out.Title != "上午好" {
		t.Fatalf("title=%s", out.Title)
	}
}
