package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// HourlyNotifyUsecase 组装定时每小时系统通知。
type HourlyNotifyUsecase struct {
	cfg config.Config
}

func NewHourlyNotifyUsecase(cfg config.Config) *HourlyNotifyUsecase {
	return &HourlyNotifyUsecase{cfg: cfg}
}

// BuildPushInput 为单个用户构建推送入参。
func (u *HourlyNotifyUsecase) BuildPushInput(userID string, slot time.Time) RealtimePushInput {
	loc := u.cfg.Scheduler.Location()
	slot = slot.In(loc).Truncate(time.Hour)
	scheduleSlot := slot.Format(time.RFC3339)
	campaignID := entity.CampaignIDFromSlot(scheduleSlot)

	hn := u.cfg.Scheduler.HourlyNotify
	title, body := resolveHourlyTemplate(hn, slot.Hour())
	expiresAt := slot.Add(time.Duration(hn.ExpiresMinutesOrDefault()) * time.Minute).UnixMilli()

	var action *entity.SysNotifyAction
	if hn.Action.Type != "" || hn.Action.Route != "" {
		action = &entity.SysNotifyAction{
			Type:   hn.Action.Type,
			Route:  hn.Action.Route,
			Params: hn.Action.Params,
			URL:    hn.Action.URL,
		}
	}

	payload := entity.NewScheduledSysNotify(entity.ScheduledSysNotifyOpts{
		Title:        title,
		Body:         body,
		CampaignID:   campaignID,
		ScheduleSlot: scheduleSlot,
		ExpiresAt:    expiresAt,
		Action:       action,
	})

	return RealtimePushInput{
		UserID: userID,
		Topic:  entity.TopicSysNotify,
		Title:  payload.Title,
		Body:   payload.Body,
		Name:   entity.EventSysNotifyShow,
		Extra:  payload.ToExtraMap(),
	}
}

// DedupTaskID 生成 Asynq 幂等任务 ID。
func (u *HourlyNotifyUsecase) DedupTaskID(userID string, slot time.Time) string {
	loc := u.cfg.Scheduler.Location()
	slot = slot.In(loc).Truncate(time.Hour)
	campaignID := entity.CampaignIDFromSlot(slot.Format(time.RFC3339))
	return fmt.Sprintf("%s:%s", campaignID, userID)
}

func resolveHourlyTemplate(hn config.HourlyNotifyConfig, hour int) (string, string) {
	switch hour {
	case 10:
		return "上午好", "新的一天开始了，查看今日动态"
	case 19:
		return "今日小结", "今日内容已更新，点击查看"
	default:
		title := hn.TitleTemplate
		if title == "" {
			title = "整点提醒"
		}
		body := hn.BodyTemplate
		if body == "" {
			body = "现在是 {{hour}}:00，{{message}}"
		}
		msg := hn.DefaultMessage
		if msg == "" {
			msg = "别错过重要消息"
		}
		body = strings.ReplaceAll(body, "{{hour}}", fmt.Sprintf("%02d", hour))
		body = strings.ReplaceAll(body, "{{message}}", msg)
		return title, body
	}
}
