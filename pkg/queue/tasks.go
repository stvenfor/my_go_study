// tasks.go 定义 Asynq 任务类型与 payload 结构。
package queue

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const (
	TypeRealtimePushNotify    = "realtime:push_notify"
	TypeScheduledBroadcastNotify = "scheduled:broadcast_notify"
	TypeSendSMS               = "sms:send"
	TypeJPushRegister         = "jpush:register"
)

// PushNotifyPayload Realtime 推送任务载荷。
type PushNotifyPayload struct {
	UserID string         `json:"userId"`
	Topic  string         `json:"topic"`
	Title  string         `json:"title"`
	Body   string         `json:"body"`
	Name   string         `json:"name"`
	Extra  map[string]any `json:"extra,omitempty"`
}

// SendSMSPayload 短信发送任务载荷（生产 OTP 预留）。
type SendSMSPayload struct {
	Phone   string `json:"phone"`
	Message string `json:"message"`
}

// JPushRegisterPayload 极光推送注册任务载荷（预留）。
type JPushRegisterPayload struct {
	UserID     string `json:"userId"`
	DeviceID   string `json:"deviceId"`
	Platform   string `json:"platform"`
	RegisterID string `json:"registerId"`
}

// BroadcastNotifyPayload 定时广播任务载荷。
type BroadcastNotifyPayload struct {
	ScheduleSlot string `json:"scheduleSlot,omitempty"`
}

// NewBroadcastNotifyTask 创建定时广播任务。
func NewBroadcastNotifyTask(p BroadcastNotifyPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal broadcast payload: %w", err)
	}
	return asynq.NewTask(TypeScheduledBroadcastNotify, data), nil
}

// ParseBroadcastNotifyPayload 解析定时广播任务载荷。
func ParseBroadcastNotifyPayload(t *asynq.Task) (BroadcastNotifyPayload, error) {
	var p BroadcastNotifyPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return BroadcastNotifyPayload{}, fmt.Errorf("unmarshal broadcast payload: %w", err)
	}
	return p, nil
}

// NewPushNotifyTask 创建 Realtime 推送任务。
func NewPushNotifyTask(p PushNotifyPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal push payload: %w", err)
	}
	return asynq.NewTask(TypeRealtimePushNotify, data), nil
}

// ParsePushNotifyPayload 解析 Realtime 推送任务载荷。
func ParsePushNotifyPayload(t *asynq.Task) (PushNotifyPayload, error) {
	var p PushNotifyPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return PushNotifyPayload{}, fmt.Errorf("unmarshal push payload: %w", err)
	}
	return p, nil
}

// NewSendSMSTask 创建短信发送任务。
func NewSendSMSTask(p SendSMSPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal sms payload: %w", err)
	}
	return asynq.NewTask(TypeSendSMS, data), nil
}

// NewJPushRegisterTask 创建极光推送注册任务。
func NewJPushRegisterTask(p JPushRegisterPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal jpush payload: %w", err)
	}
	return asynq.NewTask(TypeJPushRegister, data), nil
}
