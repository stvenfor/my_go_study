// casbin.go 基于 GORM 适配器的 Casbin Enforcer 封装。
//
// 提供 RBAC 风格的访问控制：sub(用户/角色) 对 obj(资源) 执行 act(操作) 是否被允许。
// 策略持久化到 casbin_rule 表（由 gorm-adapter 自动建表/迁移）。
//
// 与 app/internal/access 自建 RBAC 并行：老业务保持原有 SQL 实现，
// 新业务可优先通过本包的 Enforcer 进行授权决策。
package authz

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// Enforcer 包装 casbin.Enforcer，提供更简洁的 Go 风格 API。
type Enforcer struct {
	e *casbin.Enforcer
}

// modelConfPath 返回本包内 model.conf 的绝对路径，避免依赖当前工作目录。
func modelConfPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "model.conf"
	}
	return filepath.Join(filepath.Dir(file), "model.conf")
}

// NewEnforcer 基于 GORM 连接创建 Casbin Enforcer，并加载 model.conf 与策略。
// db 传入 core/database.NewPostgres 返回的 *gorm.DB 即可。
func NewEnforcer(db *gorm.DB) (*Enforcer, error) {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("创建 Casbin GORM 适配器失败: %w", err)
	}

	e, err := casbin.NewEnforcer(modelConfPath(), adapter)
	if err != nil {
		return nil, fmt.Errorf("创建 Casbin Enforcer 失败: %w", err)
	}

	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("加载 Casbin 策略失败: %w", err)
	}

	return &Enforcer{e: e}, nil
}

// Enforce 判断 sub 对 obj 是否有权执行 act。
func (enf *Enforcer) Enforce(sub, obj, act string) (bool, error) {
	return enf.e.Enforce(sub, obj, act)
}

// EnforceMatch 对兼容旧代码的别名，等价于 Enforce。
func (enf *Enforcer) EnforceMatch(sub, obj, act string) (bool, error) {
	return enf.Enforce(sub, obj, act)
}

// AddPolicy 增加一条授权策略（sub, obj, act）。
func (enf *Enforcer) AddPolicy(sub, obj, act string) (bool, error) {
	return enf.e.AddPolicy(sub, obj, act)
}

// RemovePolicy 删除一条授权策略（sub, obj, act）。
func (enf *Enforcer) RemovePolicy(sub, obj, act string) (bool, error) {
	return enf.e.RemovePolicy(sub, obj, act)
}

// GetPolicy 返回全部策略，每条为 [sub, obj, act]。
func (enf *Enforcer) GetPolicy() ([][]string, error) {
	return enf.e.GetPolicy(), nil
}

// AddRoleForUser 将 role 角色授予 user。
func (enf *Enforcer) AddRoleForUser(user, role string) (bool, error) {
	return enf.e.AddRoleForUser(user, role)
}

// DeleteRoleForUser 撤销 user 上的 role 角色。
func (enf *Enforcer) DeleteRoleForUser(user, role string) (bool, error) {
	return enf.e.DeleteRoleForUser(user, role)
}

// GetRolesForUser 返回 user 拥有的全部角色。
func (enf *Enforcer) GetRolesForUser(user string) ([]string, error) {
	return enf.e.GetRolesForUser(user)
}

// GetUsersForRole 返回拥有 role 角色的全部用户。
func (enf *Enforcer) GetUsersForRole(role string) ([]string, error) {
	return enf.e.GetUsersForRole(role)
}

// SavePolicy 将内存策略持久化到数据库。
func (enf *Enforcer) SavePolicy() error {
	return enf.e.SavePolicy()
}
