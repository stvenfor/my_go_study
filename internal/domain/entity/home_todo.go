// home_todo.go 首页待办卡相关实体。
package entity

import "time"

const (
	WysStoreJoinApplicationTable    = "wys_store_join_application"
	WysStoreCustomerTable           = "wys_store_customer"
	WysAfterSalesAppointmentTable   = "wys_after_sales_appointment"
	WysStoreReviewOrderTable        = "wys_store_review_order"

	JoinStatusPending  int16 = 0
	JoinStatusApproved int16 = 1
	JoinStatusRejected int16 = 2

	AppointmentStatusPending int16 = 0
	AppointmentStatusDone    int16 = 1

	ReviewOrderStatusPending  int16 = 0
	ReviewOrderStatusApproved int16 = 1
	ReviewOrderStatusRejected int16 = 2

	TodoTypePartnerPending         = "partner_pending"
	TodoTypeFollowUpCustomer       = "follow_up_customer"
	TodoTypeAfterSalesAppointment  = "after_sales_appointment"
	TodoTypeOrderPendingReview     = "order_pending_review"
)

// WysStoreJoinApplication 入店申请。
type WysStoreJoinApplication struct {
	ApplicationID   int64     `json:"application_id" gorm:"primaryKey"`
	StoreID         int       `json:"store_id"`
	ApplicantUserID string    `json:"applicant_user_id" gorm:"size:64"`
	Status          int16     `json:"status"`
	ReviewedBy      *string   `json:"reviewed_by,omitempty" gorm:"size:64"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WysStoreJoinApplication) TableName() string { return WysStoreJoinApplicationTable }

// WysStoreCustomer 门店客户。
type WysStoreCustomer struct {
	CustomerID     int64      `json:"customer_id" gorm:"primaryKey"`
	StoreID        int        `json:"store_id"`
	DisplayName    string     `json:"display_name" gorm:"size:128"`
	Phone          string     `json:"phone" gorm:"size:32"`
	NextFollowUpAt *time.Time `json:"next_follow_up_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (WysStoreCustomer) TableName() string { return WysStoreCustomerTable }

// WysAfterSalesAppointment 售后预约。
type WysAfterSalesAppointment struct {
	AppointmentID   int64     `json:"appointment_id" gorm:"primaryKey"`
	StoreID         int       `json:"store_id"`
	CustomerName    string    `json:"customer_name" gorm:"size:128"`
	AppointmentDate time.Time `json:"appointment_date" gorm:"type:date"`
	Status          int16     `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WysAfterSalesAppointment) TableName() string { return WysAfterSalesAppointmentTable }

// WysStoreReviewOrder 店务审核单。
type WysStoreReviewOrder struct {
	OrderID   int64     `json:"order_id" gorm:"primaryKey"`
	StoreID   int       `json:"store_id"`
	Title     string    `json:"title" gorm:"size:256"`
	Status    int16     `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysStoreReviewOrder) TableName() string { return WysStoreReviewOrderTable }

// HomeTodoCard 首页待办卡（聚合视图）。
type HomeTodoCard struct {
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Subtitle    string  `json:"subtitle"`
	ActionLabel string  `json:"action_label"`
	ActionRoute string  `json:"action_route"`
	Count       int64   `json:"count"`
	ImageURL    *string `json:"image_url,omitempty"`
}
