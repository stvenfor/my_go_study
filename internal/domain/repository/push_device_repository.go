package repository

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// PushDeviceRepository 极光设备登记。
type PushDeviceRepository interface {
	Upsert(ctx context.Context, device entity.WysPushDevice) (*entity.WysPushDevice, error)
}
