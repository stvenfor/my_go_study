// user.go 定义用户领域实体，不依赖外层框架实现细节。
package entity

import "time"

// User 用户实体，映射 users 表。
type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Email     string    `gorm:"uniqueIndex;size:128;not null" json:"email"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定 GORM 表名。
func (User) TableName() string {
	return "users"
}
