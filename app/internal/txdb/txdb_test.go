package txdb

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

func TestDBReturnsDefaultWhenNoTx(t *testing.T) {
	var def *gorm.DB // nil 也照样能断言「取回的就是默认句柄」
	if got := DB(context.Background(), def); got != def {
		t.Fatalf("无事务时应回落到默认句柄")
	}
}

func TestWithRoundTrip(t *testing.T) {
	tx := &gorm.DB{Config: &gorm.Config{}}
	ctx := With(context.Background(), tx)
	if got := DB(ctx, nil); got != tx {
		t.Fatal("ctx 上的事务句柄没取到")
	}
	// 嵌套：内层 ctx 覆盖外层，取到的是最后一次放入的句柄。
	inner := &gorm.DB{Config: &gorm.Config{}}
	if got := DB(With(ctx, inner), nil); got != inner {
		t.Fatal("嵌套事务句柄应取到最内层")
	}
}

func TestWithIgnoresNilTx(t *testing.T) {
	def := &gorm.DB{Config: &gorm.Config{}}
	// 有人误放 nil 时不能把默认句柄顶掉。
	ctx := With(context.Background(), nil)
	if got := DB(ctx, def); got != def {
		t.Fatal("nil 事务句柄应被忽略")
	}
}
