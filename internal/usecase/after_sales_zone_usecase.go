package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrAfterSalesRecordNotFound   = repository.ErrAfterSalesRecordNotFound
	ErrAfterSalesNoStore          = repository.ErrAfterSalesNoStore
	ErrAfterSalesNotMember        = repository.ErrAfterSalesNotMember
	ErrAfterSalesBadKind          = repository.ErrAfterSalesBadKind
	ErrAfterSalesBadCustomer      = repository.ErrAfterSalesBadCustomer
	ErrAfterSalesBadAppointment   = repository.ErrAfterSalesBadAppointment
	ErrAfterSalesAppointmentTaken = repository.ErrAfterSalesAppointmentTaken
	ErrAfterSalesForbidden        = repository.ErrAfterSalesForbidden
	ErrAfterSalesBadTitle         = errors.New("标题不能为空")
	ErrAfterSalesBadDate          = errors.New("服务日期无效")
)

// AfterSalesZoneUsecase 售后专区。
type AfterSalesZoneUsecase struct {
	repo   repository.AfterSalesZoneRepository
	access *AccessUsecase
	now    func() time.Time
}

func NewAfterSalesZoneUsecase(repo repository.AfterSalesZoneRepository, access *AccessUsecase) *AfterSalesZoneUsecase {
	return &AfterSalesZoneUsecase{repo: repo, access: access, now: time.Now}
}

// CreateAfterSalesRecordInput 新建记录入参。
type CreateAfterSalesRecordInput struct {
	AppointmentID  *int64
	CustomerID     *int64
	CustomerUserID *string
	CustomerName   string
	CustomerPhone  string
	PlateNo        string
	Mileage        *int
	ServiceKind    string
	Title          string
	Content        string
	ServiceDate    string // YYYY-MM-DD
}

// AfterSalesRecordDTO 对外读模型。
type AfterSalesRecordDTO struct {
	RecordID         int64   `json:"record_id"`
	StoreID          int     `json:"store_id"`
	AppointmentID    *int64  `json:"appointment_id,omitempty"`
	CustomerID       *int64  `json:"customer_id,omitempty"`
	CustomerUserID   *string `json:"customer_user_id,omitempty"`
	CustomerName     string  `json:"customer_name"`
	CustomerPhone    string  `json:"customer_phone"`
	PlateNo          string  `json:"plate_no"`
	Mileage          *int    `json:"mileage,omitempty"`
	ServiceKind      int16   `json:"service_kind"`
	ServiceKindLabel string  `json:"service_kind_label"`
	Title            string  `json:"title"`
	Content          string  `json:"content"`
	ServiceDate      string  `json:"service_date"`
	CreatedBy        string  `json:"created_by"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	CanCreate        bool    `json:"-"` // 仅列表包装用，不进单条
}

func (u *AfterSalesZoneUsecase) List(ctx context.Context, actorID string, page, size int) ([]AfterSalesRecordDTO, int64, bool, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	storeID, isMember, err := u.memberOfCurrentStore(ctx, actorID)
	if err != nil {
		return nil, 0, false, err
	}
	var rows []entity.WysAfterSalesRecord
	var total int64
	if isMember {
		rows, total, err = u.repo.ListByStore(ctx, storeID, offset, size)
	} else {
		rows, total, err = u.repo.ListByCustomerUser(ctx, actorID, offset, size)
	}
	if err != nil {
		return nil, 0, false, err
	}
	out := make([]AfterSalesRecordDTO, 0, len(rows))
	for i := range rows {
		out = append(out, toAfterSalesRecordDTO(&rows[i]))
	}
	return out, total, isMember, nil
}

func (u *AfterSalesZoneUsecase) Get(ctx context.Context, actorID string, recordID int64) (*AfterSalesRecordDTO, error) {
	row, err := u.repo.GetByID(ctx, recordID)
	if err != nil {
		return nil, err
	}
	ok, err := u.canReadRecord(ctx, actorID, row)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAfterSalesRecordNotFound
	}
	dto := toAfterSalesRecordDTO(row)
	return &dto, nil
}

func (u *AfterSalesZoneUsecase) Create(ctx context.Context, actorID string, in CreateAfterSalesRecordInput) (*AfterSalesRecordDTO, error) {
	storeID, err := u.requireMemberStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	kind, ok := entity.ParseServiceKind(strings.TrimSpace(in.ServiceKind))
	if !ok {
		return nil, ErrAfterSalesBadKind
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrAfterSalesBadTitle
	}
	serviceDate, err := parseServiceDate(in.ServiceDate)
	if err != nil {
		return nil, ErrAfterSalesBadDate
	}

	name := strings.TrimSpace(in.CustomerName)
	phone := strings.TrimSpace(in.CustomerPhone)
	var customerID *int64
	var appointmentID *int64
	if in.AppointmentID != nil && *in.AppointmentID > 0 {
		appt, err := u.repo.GetAppointment(ctx, *in.AppointmentID)
		if err != nil {
			return nil, err
		}
		if appt.StoreID != storeID || appt.Status != entity.AppointmentStatusPending {
			return nil, ErrAfterSalesBadAppointment
		}
		appointmentID = in.AppointmentID
		if name == "" {
			name = appt.CustomerName
		}
	}
	if in.CustomerID != nil && *in.CustomerID > 0 {
		cust, err := u.repo.GetStoreCustomer(ctx, storeID, *in.CustomerID)
		if err != nil {
			return nil, err
		}
		customerID = &cust.CustomerID
		if name == "" {
			name = cust.DisplayName
		}
		if phone == "" {
			phone = cust.Phone
		}
	}
	if name == "" || phone == "" {
		return nil, ErrAfterSalesBadCustomer
	}

	var custUser *string
	if in.CustomerUserID != nil {
		s := strings.TrimSpace(*in.CustomerUserID)
		if s != "" {
			custUser = &s
		}
	}

	now := u.now()
	row := &entity.WysAfterSalesRecord{
		StoreID:        storeID,
		AppointmentID:  appointmentID,
		CustomerID:     customerID,
		CustomerUserID: custUser,
		CustomerName:   name,
		CustomerPhone:  phone,
		PlateNo:        strings.TrimSpace(in.PlateNo),
		Mileage:        in.Mileage,
		ServiceKind:    kind,
		Title:          title,
		Content:        strings.TrimSpace(in.Content),
		ServiceDate:    serviceDate,
		CreatedBy:      actorID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := u.repo.CreateRecord(ctx, row); err != nil {
		return nil, err
	}
	dto := toAfterSalesRecordDTO(row)
	return &dto, nil
}

func (u *AfterSalesZoneUsecase) ListPendingAppointments(ctx context.Context, actorID string) ([]entity.WysAfterSalesAppointment, error) {
	storeID, err := u.requireMemberStore(ctx, actorID)
	if err != nil {
		return nil, err
	}
	return u.repo.ListPendingAppointments(ctx, storeID, shanghaiDate(u.now()))
}

func ParseAfterSalesRecordID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrAfterSalesRecordNotFound
	}
	return id, nil
}

func (u *AfterSalesZoneUsecase) requireMemberStore(ctx context.Context, actorID string) (int, error) {
	storeID, isMember, err := u.memberOfCurrentStore(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if storeID == 0 {
		return 0, ErrAfterSalesNoStore
	}
	if !isMember {
		return 0, ErrAfterSalesNotMember
	}
	return storeID, nil
}

func (u *AfterSalesZoneUsecase) memberOfCurrentStore(ctx context.Context, actorID string) (storeID int, isMember bool, err error) {
	if u.access == nil {
		return 0, false, ErrAfterSalesNoStore
	}
	cur, err := u.access.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return 0, false, err
	}
	if cur == nil || *cur <= 0 {
		return 0, false, nil
	}
	storeID = *cur
	member, err := u.access.repo.GetMember(ctx, actorID, storeID)
	if err != nil {
		if errors.Is(err, repository.ErrAccessMemberNotFound) {
			return storeID, false, nil
		}
		return 0, false, err
	}
	return storeID, member != nil, nil
}

func (u *AfterSalesZoneUsecase) canReadRecord(ctx context.Context, actorID string, row *entity.WysAfterSalesRecord) (bool, error) {
	if row.CustomerUserID != nil && *row.CustomerUserID == actorID {
		return true, nil
	}
	if u.access == nil {
		return false, nil
	}
	member, err := u.access.repo.GetMember(ctx, actorID, row.StoreID)
	if err != nil {
		if errors.Is(err, repository.ErrAccessMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return member != nil, nil
}

func toAfterSalesRecordDTO(row *entity.WysAfterSalesRecord) AfterSalesRecordDTO {
	return AfterSalesRecordDTO{
		RecordID:         row.RecordID,
		StoreID:          row.StoreID,
		AppointmentID:    row.AppointmentID,
		CustomerID:       row.CustomerID,
		CustomerUserID:   row.CustomerUserID,
		CustomerName:     row.CustomerName,
		CustomerPhone:    row.CustomerPhone,
		PlateNo:          row.PlateNo,
		Mileage:          row.Mileage,
		ServiceKind:      row.ServiceKind,
		ServiceKindLabel: entity.ServiceKindLabel(row.ServiceKind),
		Title:            row.Title,
		Content:          row.Content,
		ServiceDate:      row.ServiceDate.Format("2006-01-02"),
		CreatedBy:        row.CreatedBy,
		CreatedAt:        row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        row.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func parseServiceDate(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, errors.New("empty")
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	t, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}
