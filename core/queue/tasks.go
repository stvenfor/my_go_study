// tasks.go 定义任务类型与 envelope（RabbitMQ JSON）。
package queue

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const (
	TypeRealtimePushNotify       = "realtime:push_notify"
	TypeScheduledBroadcastNotify = "scheduled:broadcast_notify"
	TypeSendSMS                  = "sms:send"
	TypeJPushRegister            = "jpush:register"
	// TypePaymentQueryPending 主动查单兜底：扫超时未核销的渠道流水并向渠道查单。
	TypePaymentQueryPending = "payment:query_pending"
	// TypePaymentReconcile T+1 对账：比对本日流水与渠道事实，差异落表。
	TypePaymentReconcile = "payment:reconcile"
)

// Envelope 入队/消费统一信封。
type Envelope struct {
	Type    string          `json:"type"`
	TaskID  string          `json:"task_id"`
	Payload json.RawMessage `json:"payload"`
}

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

// PaymentQueryPendingPayload 主动查单任务载荷。阈值/批量一律取 worker 侧
// scheduler.payment.*，不放载荷：改配置不用重放旧任务，避免队列里留着过期参数。
type PaymentQueryPendingPayload struct{}

// PaymentReconcilePayload T+1 对账任务载荷。
// Date 为空表示「昨天」；显式传 2026-11-03 可人工补跑某日。
type PaymentReconcilePayload struct {
	Date string `json:"date,omitempty"`
}

func newEnvelope(taskType string, payload any, taskID string) (Envelope, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("marshal payload: %w", err)
	}
	if taskID == "" {
		taskID = uuid.NewString()
	}
	return Envelope{Type: taskType, TaskID: taskID, Payload: data}, nil
}

// NewBroadcastNotifyTask 创建定时广播任务。
func NewBroadcastNotifyTask(p BroadcastNotifyPayload) (Envelope, error) {
	return newEnvelope(TypeScheduledBroadcastNotify, p, "")
}

// ParseBroadcastNotifyPayload 解析定时广播任务载荷。
func ParseBroadcastNotifyPayload(payload []byte) (BroadcastNotifyPayload, error) {
	var p BroadcastNotifyPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return BroadcastNotifyPayload{}, fmt.Errorf("unmarshal broadcast payload: %w", err)
	}
	return p, nil
}

// NewPushNotifyTask 创建 Realtime 推送任务。
func NewPushNotifyTask(p PushNotifyPayload) (Envelope, error) {
	return newEnvelope(TypeRealtimePushNotify, p, "")
}

// ParsePushNotifyPayload 解析 Realtime 推送任务载荷。
func ParsePushNotifyPayload(payload []byte) (PushNotifyPayload, error) {
	var p PushNotifyPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return PushNotifyPayload{}, fmt.Errorf("unmarshal push payload: %w", err)
	}
	return p, nil
}

// NewSendSMSTask 创建短信发送任务。
func NewSendSMSTask(p SendSMSPayload) (Envelope, error) {
	return newEnvelope(TypeSendSMS, p, "")
}

// NewJPushRegisterTask 创建极光推送注册任务。
func NewJPushRegisterTask(p JPushRegisterPayload) (Envelope, error) {
	return newEnvelope(TypeJPushRegister, p, "")
}

// NewPaymentQueryPendingTask 创建主动查单任务。
func NewPaymentQueryPendingTask(p PaymentQueryPendingPayload) (Envelope, error) {
	return newEnvelope(TypePaymentQueryPending, p, "")
}

// ParsePaymentQueryPendingPayload 解析主动查单任务载荷。
func ParsePaymentQueryPendingPayload(payload []byte) (PaymentQueryPendingPayload, error) {
	var p PaymentQueryPendingPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return PaymentQueryPendingPayload{}, fmt.Errorf("unmarshal payment query payload: %w", err)
	}
	return p, nil
}

// NewPaymentReconcileTask 创建 T+1 对账任务。
func NewPaymentReconcileTask(p PaymentReconcilePayload) (Envelope, error) {
	return newEnvelope(TypePaymentReconcile, p, "")
}

// ParsePaymentReconcilePayload 解析对账任务载荷。
func ParsePaymentReconcilePayload(payload []byte) (PaymentReconcilePayload, error) {
	var p PaymentReconcilePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return PaymentReconcilePayload{}, fmt.Errorf("unmarshal payment reconcile payload: %w", err)
	}
	return p, nil
}
