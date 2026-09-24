package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed sql/push_device_schema.sql
var pushDeviceSchemaSQL string

// PushDeviceRepository 极光设备 Postgres。
type PushDeviceRepository struct {
	db *gorm.DB
}

func NewPushDeviceRepository(db *gorm.DB) *PushDeviceRepository {
	return &PushDeviceRepository{db: db}
}

// EnsurePushDeviceSchema 幂等建表。
func EnsurePushDeviceSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	if err := db.Exec(pushDeviceSchemaSQL).Error; err != nil {
		return fmt.Errorf("push device schema: %w", err)
	}
	return db.AutoMigrate(&entity.WysPushDevice{})
}

func (r *PushDeviceRepository) Upsert(ctx context.Context, device entity.WysPushDevice) (*entity.WysPushDevice, error) {
	if strings.TrimSpace(device.UserID) == "" {
		return nil, fmt.Errorf("user_id required")
	}
	if strings.TrimSpace(device.DeviceID) == "" {
		device.DeviceID = "default"
	}
	now := time.Now().UTC()
	device.UpdatedAt = now
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}
	device.Platform = normalizePlatform(device.Platform)

	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"platform", "registration_id", "alias", "mock", "updated_at",
		}),
	}).Create(&device).Error
	if err != nil {
		return nil, err
	}
	var out entity.WysPushDevice
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND device_id = ?", device.UserID, device.DeviceID).
		First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func normalizePlatform(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "ios":
		return "ios"
	case "android":
		return "android"
	case "harmony", "ohos", "harmonyos":
		return "harmony"
	default:
		return "unknown"
	}
}
