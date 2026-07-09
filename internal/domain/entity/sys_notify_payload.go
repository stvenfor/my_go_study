package entity

import (
	"strings"

	"github.com/google/uuid"
)

// SysNotifyAction 通知点击行为。
type SysNotifyAction struct {
	Type   string         `json:"type,omitempty"`
	Route  string         `json:"route,omitempty"`
	Params map[string]any `json:"params,omitempty"`
	URL    string         `json:"url,omitempty"`
}

// SysNotifyPayload 系统通知业务载荷（payload 层）。
type SysNotifyPayload struct {
	Name         string           `json:"name"`
	NotifyID     string           `json:"notifyId"`
	Title        string           `json:"title"`
	Body         string           `json:"body"`
	Category     string           `json:"category,omitempty"`
	Priority     string           `json:"priority,omitempty"`
	CampaignID   string           `json:"campaignId,omitempty"`
	ScheduleSlot string           `json:"scheduleSlot,omitempty"`
	MessageType  string           `json:"messageType,omitempty"`
	Locale       string           `json:"locale,omitempty"`
	Silent       bool             `json:"silent,omitempty"`
	ExpiresAt    int64            `json:"expiresAt,omitempty"`
	Action       *SysNotifyAction `json:"action,omitempty"`
	ImageURL     string           `json:"imageUrl,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// ScheduledSysNotifyOpts 定时通知构建参数。
type ScheduledSysNotifyOpts struct {
	Title        string
	Body         string
	CampaignID   string
	ScheduleSlot string
	ExpiresAt    int64
	Action       *SysNotifyAction
	Metadata     map[string]any
}

// NewScheduledSysNotify 构建定时系统通知载荷。
func NewScheduledSysNotify(opts ScheduledSysNotifyOpts) SysNotifyPayload {
	meta := opts.Metadata
	if meta == nil {
		meta = map[string]any{"source": "scheduler", "version": "1"}
	}
	return SysNotifyPayload{
		Name:         EventSysNotifyShow,
		NotifyID:     uuid.NewString(),
		Title:        opts.Title,
		Body:         opts.Body,
		Category:     "scheduled",
		Priority:     "normal",
		CampaignID:   opts.CampaignID,
		ScheduleSlot: opts.ScheduleSlot,
		MessageType:  "hourly_digest",
		Locale:       "zh-CN",
		Silent:       false,
		ExpiresAt:    opts.ExpiresAt,
		Action:       opts.Action,
		Metadata:     meta,
	}
}

// ToExtraMap 返回扩展字段（不含 name/notifyId/title/body，由 DeliverPush 写入）。
func (p SysNotifyPayload) ToExtraMap() map[string]any {
	out := map[string]any{
		"category":     p.Category,
		"priority":     p.Priority,
		"campaignId":   p.CampaignID,
		"scheduleSlot": p.ScheduleSlot,
		"messageType":  p.MessageType,
		"locale":       p.Locale,
		"silent":       p.Silent,
	}
	if p.ExpiresAt > 0 {
		out["expiresAt"] = p.ExpiresAt
	}
	if p.ImageURL != "" {
		out["imageUrl"] = p.ImageURL
	}
	if p.Action != nil {
		out["action"] = p.Action
	}
	if p.Metadata != nil {
		out["metadata"] = p.Metadata
	}
	return out
}

// CampaignIDFromSlot 由计划槽位生成批次 ID。
func CampaignIDFromSlot(scheduleSlot string) string {
	compact := strings.NewReplacer(":", "", "-", "", "T", "-", "+", "").Replace(scheduleSlot)
	return "hourly-" + compact
}
