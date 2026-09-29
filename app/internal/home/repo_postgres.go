package home

import (
	"context"
	"errors"
	"time"

	"app/internal/shared"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"model/entity"
)

// PostgresRepository 首页待办。
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository 创建。
func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateJoinApplication(ctx context.Context, app *entity.WysStoreJoinApplication) error {
	err := r.db.WithContext(ctx).Create(app).Error
	if err != nil {
		if shared.IsUniqueViolation(err) {
			return ErrJoinApplicationDuplicate
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) GetJoinApplication(ctx context.Context, applicationID int64) (*entity.WysStoreJoinApplication, error) {
	var app entity.WysStoreJoinApplication
	err := r.db.WithContext(ctx).Where("application_id = ?", applicationID).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrJoinApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func (r *PostgresRepository) ListPendingJoinApplications(ctx context.Context, storeID int) ([]entity.WysStoreJoinApplication, error) {
	var rows []entity.WysStoreJoinApplication
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ?", storeID, entity.JoinStatusPending).
		Order("created_at ASC, application_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) CountPendingJoinApplications(ctx context.Context, storeID int) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysStoreJoinApplication{}).
		Where("store_id = ? AND status = ?", storeID, entity.JoinStatusPending).
		Count(&n).Error
	return n, err
}

func (r *PostgresRepository) UpdateJoinApplicationStatus(ctx context.Context, applicationID int64, status int16, reviewedBy string) error {
	res := r.db.WithContext(ctx).Model(&entity.WysStoreJoinApplication{}).
		Where("application_id = ? AND status = ?", applicationID, entity.JoinStatusPending).
		Updates(map[string]interface{}{
			"status":      status,
			"reviewed_by": reviewedBy,
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJoinApplicationNotPending
	}
	return nil
}

func (r *PostgresRepository) ListOverdueCustomers(ctx context.Context, storeID int, now time.Time) ([]entity.WysStoreCustomer, error) {
	var rows []entity.WysStoreCustomer
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND next_follow_up_at IS NOT NULL AND next_follow_up_at <= ?", storeID, now).
		Order("next_follow_up_at ASC, customer_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) CountOverdueCustomers(ctx context.Context, storeID int, now time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysStoreCustomer{}).
		Where("store_id = ? AND next_follow_up_at IS NOT NULL AND next_follow_up_at <= ?", storeID, now).
		Count(&n).Error
	return n, err
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

func (r *PostgresRepository) CountPendingAppointments(ctx context.Context, storeID int, today time.Time) (int64, error) {
	day := today.Format("2006-01-02")
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysAfterSalesAppointment{}).
		Where("store_id = ? AND status = ? AND appointment_date >= ?", storeID, entity.AppointmentStatusPending, day).
		Count(&n).Error
	return n, err
}

func (r *PostgresRepository) ListPendingReviewOrders(ctx context.Context, storeID int) ([]entity.WysStoreReviewOrder, error) {
	var rows []entity.WysStoreReviewOrder
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ?", storeID, entity.ReviewOrderStatusPending).
		Order("created_at ASC, order_id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) CountPendingReviewOrders(ctx context.Context, storeID int) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysStoreReviewOrder{}).
		Where("store_id = ? AND status = ?", storeID, entity.ReviewOrderStatusPending).
		Count(&n).Error
	return n, err
}

func (r *PostgresRepository) CountPendingUsedCarOrders(ctx context.Context, storeID int) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND status = ?", storeID, entity.UsedCarOrderPendingReview).
		Count(&n).Error
	return n, err
}

func (r *PostgresRepository) EnsureSeed(ctx context.Context, storeID int, applicantUserID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 仅当申请人真实存在时才种入店申请，避免确认时 UserExists 失败。
		if applicantUserID != "" {
			var userN int64
			if err := tx.Table("users").Where("user_id = ? AND deleted_at IS NULL", applicantUserID).Count(&userN).Error; err != nil {
				return err
			}
			if userN > 0 {
				var joinN int64
				if err := tx.Model(&entity.WysStoreJoinApplication{}).
					Where("store_id = ? AND status = ?", storeID, entity.JoinStatusPending).
					Count(&joinN).Error; err != nil {
					return err
				}
				if joinN == 0 {
					app := entity.WysStoreJoinApplication{
						StoreID:         storeID,
						ApplicantUserID: applicantUserID,
						Status:          entity.JoinStatusPending,
					}
					if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&app).Error; err != nil {
						return err
					}
				}
			}
		}

		var custN int64
		if err := tx.Model(&entity.WysStoreCustomer{}).Where("store_id = ?", storeID).Count(&custN).Error; err != nil {
			return err
		}
		if custN == 0 {
			past := time.Now().Add(-24 * time.Hour)
			row := entity.WysStoreCustomer{
				StoreID:        storeID,
				DisplayName:    "种子客户-待跟进",
				Phone:          "13800001111",
				NextFollowUpAt: &past,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		var apptN int64
		if err := tx.Model(&entity.WysAfterSalesAppointment{}).Where("store_id = ?", storeID).Count(&apptN).Error; err != nil {
			return err
		}
		if apptN == 0 {
			loc, err := time.LoadLocation("Asia/Shanghai")
			if err != nil {
				loc = time.FixedZone("CST", 8*3600)
			}
			local := time.Now().In(loc)
			day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
			row := entity.WysAfterSalesAppointment{
				StoreID:         storeID,
				CustomerName:    "种子预约客户",
				AppointmentDate: day,
				Status:          entity.AppointmentStatusPending,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		var ordN int64
		if err := tx.Model(&entity.WysStoreReviewOrder{}).Where("store_id = ?", storeID).Count(&ordN).Error; err != nil {
			return err
		}
		if ordN == 0 {
			row := entity.WysStoreReviewOrder{
				StoreID: storeID,
				Title:   "种子店务审核单",
				Status:  entity.ReviewOrderStatusPending,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
