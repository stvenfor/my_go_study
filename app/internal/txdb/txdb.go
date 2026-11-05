// Package txdb 提供「把事务句柄放进 context」的最小传递机制（parity-p0-3）。
//
// 为什么不用参数传 *gorm.DB：资金链路的编排者（mall.Service）调的是 wallet/points 的 Service，
// Service 又调各自的 Repository，加一层就要改三个包的签名。context 里带一句柄，
// 仓储层一律 `txdb.DB(ctx, r.db)` —— 有事务就用，没有就用默认，调用方无感。
//
// 只服务资金路径，不做全域推广（那是 P2-9）；事务里禁止外部 HTTP/ES/MQ 调用。
package txdb

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// Begin 在 db 上开一个事务，把带事务句柄的 ctx 交给 fn：fn 返回 nil 提交，返回 error 整体回滚。
// 嵌套调用（fn 里的仓储又开自己的事务）在 GORM 里落为 SAVEPOINT，内层失败只回滚到保存点。
func Begin(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(With(ctx, tx))
	})
}

// With 把事务句柄挂到 ctx 上。
func With(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// DB 返回 ctx 上的事务句柄；没有则返回 def。
// 用法：把仓储里的 r.db.WithContext(ctx) 换成 DB(ctx, r.db).WithContext(ctx)。
func DB(ctx context.Context, def *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return def
}
