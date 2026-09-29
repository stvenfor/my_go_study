// =============================================================================
// 文件：push_enqueuer.go
// 作用：把 pkg/queue 的 RabbitMQ Client 适配成 usecase.PushEnqueuer 端口。
//
// 【为什么需要这个适配器？】
//   usecase 层用依赖倒置定义了 PushEnqueuer 端口（入参是 usecase.RealtimePushInput），
//   pkg/queue 是实现方。若让 Client 直接实现该接口，基建层就得 import 业务层，
//   形成 usecase → pkg/queue → usecase 的循环，模块化时无法拆分。
//
//   所以转换放在业务侧：本包（repository 层）→ pkg/queue 单向依赖。
// =============================================================================
package queue

import (
	"context"

	"app/internal/usecase"
	pkgqueue "core/queue"
)

// PushEnqueuer 将 usecase.RealtimePushInput 转换为队列载荷并入队。
type PushEnqueuer struct {
	client *pkgqueue.Client
}

// NewPushEnqueuer 创建适配器。client 为 nil 时返回 nil，调用方按「未启用队列」处理。
func NewPushEnqueuer(client *pkgqueue.Client) *PushEnqueuer {
	if client == nil {
		return nil
	}
	return &PushEnqueuer{client: client}
}

// EnqueueRealtimePush 实现 usecase.PushEnqueuer。
func (e *PushEnqueuer) EnqueueRealtimePush(ctx context.Context, input usecase.RealtimePushInput) (string, error) {
	return e.client.EnqueuePushNotify(ctx, toPayload(input))
}

// toPayload 业务输入 → 队列载荷。
//
// 两者字段同名同序，但目前是两个独立的类型：业务侧改名不该改协议。
func toPayload(input usecase.RealtimePushInput) pkgqueue.PushNotifyPayload {
	return pkgqueue.PushNotifyPayload{
		UserID: input.UserID,
		Topic:  input.Topic,
		Title:  input.Title,
		Body:   input.Body,
		Name:   input.Name,
		Extra:  input.Extra,
	}
}
