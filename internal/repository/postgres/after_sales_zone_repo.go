package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

//go:embed sql/after_sales_zone_schema.sql
var afterSalesZoneSchemaSQL string

// AfterSalesZoneRepository 售后专区 Postgres。
type AfterSalesZoneRepository struct {
	db *gorm.DB
}

func NewAfterSalesZoneRepository(db *gorm.DB) *AfterSalesZoneRepository {
	return &AfterSalesZoneRepository{db: db}
}

// EnsureAfterSalesZoneSchema 幂等建表。
func EnsureAfterSalesZoneSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	if err := db.Exec(afterSalesZoneSchemaSQL).Error; err != nil {
		return fmt.Errorf("after_sales_zone schema: %w", err)
	}
	return db.AutoMigrate(&entity.WysAfterSalesRecord{})
}

func (r *AfterSalesZoneRepository) ListByStore(ctx context.Context, storeID int, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&entity.WysAfterSalesRecord{}).Where("store_id = ?", storeID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysAfterSalesRecord
	err := q.Order("created_at DESC, record_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *AfterSalesZoneRepository) ListByCustomerUser(ctx context.Context, customerUserID string, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&entity.WysAfterSalesRecord{}).Where("customer_user_id = ?", customerUserID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysAfterSalesRecord
	err := q.Order("created_at DESC, record_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *AfterSalesZoneRepository) GetByID(ctx context.Context, recordID int64) (*entity.WysAfterSalesRecord, error) {
	var row entity.WysAfterSalesRecord
	err := r.db.WithContext(ctx).Where("record_id = ?", recordID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrAfterSalesRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AfterSalesZoneRepository) GetStoreCustomer(ctx context.Context, storeID int, customerID int64) (*entity.WysStoreCustomer, error) {
	var row entity.WysStoreCustomer
	err := r.db.WithContext(ctx).
		Where("customer_id = ? AND store_id = ?", customerID, storeID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrAfterSalesBadCustomer
	}
	return &row, err
}

func (r *AfterSalesZoneRepository) GetAppointment(ctx context.Context, appointmentID int64) (*entity.WysAfterSalesAppointment, error) {
	var row entity.WysAfterSalesAppointment
	err := r.db.WithContext(ctx).Where("appointment_id = ?", appointmentID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrAfterSalesBadAppointment
	}
	return &row, err
}

func (r *AfterSalesZoneRepository) ListPendingAppointments(ctx context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error) {
	day := today.Format("2006-01-02")
	var rows []entity.WysAfterSalesAppointment
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ? AND appointment_date >= ?", storeID, entity.AppointmentStatusPending, day).
		Order("appointment_date ASC, appointment_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *AfterSalesZoneRepository) CreateRecord(ctx context.Context, row *entity.WysAfterSalesRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.AppointmentID != nil {
			var appt entity.WysAfterSalesAppointment
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("appointment_id = ?", *row.AppointmentID).
				First(&appt).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrAfterSalesBadAppointment
			}
			if err != nil {
				return err
			}
			if appt.StoreID != row.StoreID || appt.Status != entity.AppointmentStatusPending {
				return repository.ErrAfterSalesBadAppointment
			}
			res := tx.Model(&entity.WysAfterSalesAppointment{}).
				Where("appointment_id = ? AND status = ?", *row.AppointmentID, entity.AppointmentStatusPending).
				Updates(map[string]any{
					"status":     entity.AppointmentStatusDone,
					"updated_at": time.Now(),
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return repository.ErrAfterSalesBadAppointment
			}
		}
		if err := tx.Create(row).Error; err != nil {
			if isUniqueViolation(err) {
				return repository.ErrAfterSalesAppointmentTaken
			}
			return err
		}
		return nil
	})
}
