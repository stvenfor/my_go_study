package repository

import (
	"context"

	"model/entity"
)

// PushDeviceRepository 极光设备登记。
type PushDeviceRepository interface {
	Upsert(ctx context.Context, device entity.WysPushDevice) (*entity.WysPushDevice, error)
}
