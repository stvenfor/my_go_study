package entity_test

import (
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

func TestNewScheduledSysNotify(t *testing.T) {
	payload := entity.NewScheduledSysNotify(entity.ScheduledSysNotifyOpts{
		Title:        "上午好",
		Body:         "测试",
		CampaignID:   "hourly-20260710-10",
		ScheduleSlot: "2026-07-10T10:00:00+08:00",
		ExpiresAt:    1739007204000,
		Action: &entity.SysNotifyAction{
			Type:  "deeplink",
			Route: "/home",
		},
	})
	if payload.Name != entity.EventSysNotifyShow {
		t.Fatalf("name=%s", payload.Name)
	}
	if payload.NotifyID == "" {
		t.Fatal("notifyId empty")
	}
	extra := payload.ToExtraMap()
	if extra["campaignId"] != "hourly-20260710-10" {
		t.Fatalf("extra=%v", extra)
	}
	if extra["title"] != nil {
		t.Fatal("title should not be in extra")
	}
}

func TestCampaignIDFromSlot(t *testing.T) {
	got := entity.CampaignIDFromSlot("2026-07-10T11:00:00+08:00")
	if got == "" {
		t.Fatal("empty campaign id")
	}
}
