package after_sales

import (
	"context"
	"errors"
	"fmt"
	"time"

	"app/internal/shared"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"model/entity"
)

// PostgresRepository 售后专区 Postgres。
type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// EnsureSchema 幂等建表兜底。
//
// 权威 DDL 是 repository/postgres/sql/after_sales_zone_schema.sql，由
// postgres.ConsolidateLocalUsers 在每次启动时执行（同一份文件、同样的语句）。
// 域切片后该 .sql 仍留在 repository/postgres，embed 不能跨目录上溯，
// 故域内不再重复 embed，只保留 GORM AutoMigrate 兜底。
func EnsureSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db nil")
	}
	return db.AutoMigrate(&entity.WysAfterSalesRecord{})
}

func (r *PostgresRepository) ListByStore(ctx context.Context, storeID int, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&entity.WysAfterSalesRecord{}).Where("store_id = ?", storeID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysAfterSalesRecord
	err := q.Order("created_at DESC, record_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *PostgresRepository) ListByCustomerUser(ctx context.Context, customerUserID string, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&entity.WysAfterSalesRecord{}).Where("customer_user_id = ?", customerUserID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysAfterSalesRecord
	err := q.Order("created_at DESC, record_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *PostgresRepository) GetByID(ctx context.Context, recordID int64) (*entity.WysAfterSalesRecord, error) {
	var row entity.WysAfterSalesRecord
	err := r.db.WithContext(ctx).Where("record_id = ?", recordID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PostgresRepository) GetStoreCustomer(ctx context.Context, storeID int, customerID int64) (*entity.WysStoreCustomer, error) {
	var row entity.WysStoreCustomer
	err := r.db.WithContext(ctx).
		Where("customer_id = ? AND store_id = ?", customerID, storeID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBadCustomer
	}
	return &row, err
}

func (r *PostgresRepository) GetAppointment(ctx context.Context, appointmentID int64) (*entity.WysAfterSalesAppointment, error) {
	var row entity.WysAfterSalesAppointment
	err := r.db.WithContext(ctx).Where("appointment_id = ?", appointmentID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBadAppointment
	}
	return &row, err
}

func (r *PostgresRepository) ListPendingAppointments(ctx context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error) {
	day := today.Format("2006-01-02")
	var rows []entity.WysAfterSalesAppointment
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ? AND appointment_date >= ?", storeID, entity.AppointmentStatusPending, day).
		Order("appointment_date ASC, appointment_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) CreateRecord(ctx context.Context, row *entity.WysAfterSalesRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if row.AppointmentID != nil {
			var appt entity.WysAfterSalesAppointment
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("appointment_id = ?", *row.AppointmentID).
				First(&appt).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBadAppointment
			}
			if err != nil {
				return err
			}
			if appt.StoreID != row.StoreID || appt.Status != entity.AppointmentStatusPending {
				return ErrBadAppointment
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
				return ErrBadAppointment
			}
		}
		if err := tx.Create(row).Error; err != nil {
			if shared.IsUniqueViolation(err) {
				return ErrAppointmentTaken
			}
			return err
		}
		return nil
	})
}
