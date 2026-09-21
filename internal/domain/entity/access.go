// access.go 权限角色与门店成员。职务不是权限。
package entity

import "time"

const (
	RolePlatformAdmin = "platform_admin"
	RoleStoreAdmin    = "store_admin"
	RoleStoreStaff    = "store_staff"

	ScopePlatform = "platform"
	ScopeStore    = "store"

	PermStoreCreate        = "store.create"
	PermMemberWrite        = "member.write"
	PermRoleAssignStore    = "role.assign_store"
	PermRoleAssignPlatform = "role.assign_platform"
	PermProfileRead        = "profile.read"
	PermTxnReadOwn         = "transaction.read_own"
	PermTxnWriteOwn        = "transaction.write_own"
	PermTxnReadStore       = "transaction.read_store"
)

const (
	WysStoreTable       = "wys_store"
	WysStoreMemberTable = "wys_store_member"
	RoleTable           = "role"
	PermissionTable     = "permission"
	RolePermissionTable = "role_permission"
	UserRoleTable       = "user_role"
)

// WysStore 门店。
type WysStore struct {
	StoreID   int       `json:"store_id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysStore) TableName() string { return WysStoreTable }

// WysStoreMember 门店成员。一人一店一个职务。
type WysStoreMember struct {
	UserID    string    `json:"user_id" gorm:"primaryKey;size:64"`
	StoreID   int       `json:"store_id" gorm:"primaryKey"`
	Position  int16     `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysStoreMember) TableName() string { return WysStoreMemberTable }

// Role 内置权限角色。
type Role struct {
	Code  string `json:"code" gorm:"primaryKey;size:64"`
	Scope string `json:"scope"`
	Name  string `json:"name"`
}

func (Role) TableName() string { return RoleTable }

// Permission 能力码。
type Permission struct {
	Code string `json:"code" gorm:"primaryKey;size:64"`
	Name string `json:"name"`
}

func (Permission) TableName() string { return PermissionTable }

// UserRole 角色分配。平台行 StoreID 为空。
type UserRole struct {
	UserID    string    `json:"user_id" gorm:"size:64"`
	RoleCode  string    `json:"role_code" gorm:"size:64"`
	StoreID   *int      `json:"store_id,omitempty"`
	GrantedBy *string   `json:"granted_by,omitempty" gorm:"size:64"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserRole) TableName() string { return UserRoleTable }

// PermissionNeedsStore 该能力是否必须带目标门店。
func PermissionNeedsStore(code string) bool {
	switch code {
	case PermMemberWrite, PermRoleAssignStore:
		return true
	default:
		return false
	}
}
