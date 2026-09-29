// im_sync.go 门店群同步端口：跨域消费的接口与结果类型。
//
// 原先 ImStoreGroupOut / ImStoreGroupSyncer 定义在 usecase 包的
// profile_usecase.go 里，被 access 与 profile 两个域共同消费（由 im 域实现）。
// 域切片后三者不在同一包，为保证方法签名类型完全一致（Go 接口是结构化满足），
// 上提到共享包。
package shared

import "context"

// ImStoreGroupOut 门店群同步结果。
type ImStoreGroupOut struct {
	StoreID string `json:"store_id"`
	GroupID string `json:"group_id"`
}

// ImStoreGroupSyncer 门店群幂等补拉（可选；未接线时调用方按 nil 判空跳过）。
//
// 实现方：im 域的 ImGroupUsecase。消费方：access 域（入店拉人 / 离店踢人）、
// profile 域（登录切换门店后幂等入群）。
type ImStoreGroupSyncer interface {
	EnsureStoreMembership(ctx context.Context, storeID, userID string) (*ImStoreGroupOut, error)
	RemoveStoreMembership(ctx context.Context, storeID, userID string) error
}
