// money_fen_test.go 金额「元 ↔ 分」换算的单测（harden-h6b）。
//
// 覆盖的是**换算口径本身**（不连库）：真库行为见 money_fen_integration_test.go（-tags=integration）。
package transaction

import (
	"math"
	"testing"

	"model/entity"
)

func TestMoneyUnitYuanToFenAcceptsTwoDecimals(t *testing.T) {
	cases := []struct {
		yuan float64
		fen  int64
	}{
		{0, 0},
		{0.1, 10}, // 二进制真值是 0.10000000000000000555…，仍必须收 —— 否则合法金额全被拒
		{0.2, 20},
		{0.3, 30},
		{88.5, 8850}, // scripts/test_transactions_crud.sh 用的量级
		{99.9, 9990}, // 同上
		{1234.56, 123456},
		{0.07, 7}, // 0.07*100 在 float 上是 7.000000000000001
	}
	for _, c := range cases {
		got, err := entity.AmountFenFromYuan(c.yuan)
		if err != nil {
			t.Errorf("AmountFenFromYuan(%v) 报错，应接受两位小数: %v", c.yuan, err)
			continue
		}
		if got != c.fen {
			t.Errorf("AmountFenFromYuan(%v) = %d, 期望 %d", c.yuan, got, c.fen)
		}
	}
}

func TestMoneyUnitYuanToFenRejectsSubCentAndSign(t *testing.T) {
	// 这些是「静默舍入会吃掉钱」的输入：必须报错，不许 round 到 100 分。
	for _, yuan := range []float64{
		0.30000000000000004, // 0.1+0.2 的 float 累加残值
		1.001,
		1.005,
		-0.01, // 负数由 type 表达
		-1,
		math.NaN(),
		math.Inf(1),
	} {
		if fen, err := entity.AmountFenFromYuan(yuan); err == nil {
			t.Errorf("AmountFenFromYuan(%v) = %d 无错，应拒绝（>2 位小数或负数）", yuan, fen)
		}
	}
}

// TestMoneyUnitAggregateIsExact 本 Slice 存在的理由：
// 按分聚合等于 0.30，按元（float）累加等于 0.30000000000000004。
func TestMoneyUnitAggregateIsExact(t *testing.T) {
	a, err := entity.AmountFenFromYuan(0.1)
	if err != nil {
		t.Fatalf("0.1 应可换算: %v", err)
	}
	b, err := entity.AmountFenFromYuan(0.2)
	if err != nil {
		t.Fatalf("0.2 应可换算: %v", err)
	}
	var sumFen int64 = a + b
	if got := entity.YuanFromAmountFen(sumFen); got != 0.3 {
		t.Errorf("分聚合后转元 = %v，期望 0.3", got)
	}
	// 对照组：旧口径的 float 累加确实不等于 0.3。
	// 必须用**运行期**变量相加：无类型常量的运算是任意精度的，写成 `0.1+0.2 == 0.3` 编译期恒为 true。
	x, y := 0.1, 0.2
	if x+y == 0.3 {
		t.Fatal("前提变了：float64 运行期 0.1+0.2 竟然等于 0.3，本用例的反证失效")
	}
}

func TestMoneyUnitFenToYuanRoundTrip(t *testing.T) {
	for fen := int64(0); fen <= 100000; fen += 7 {
		yuan := entity.YuanFromAmountFen(fen)
		back, err := entity.AmountFenFromYuan(yuan)
		if err != nil {
			t.Fatalf("fen=%d → yuan=%v 反向换算报错: %v", fen, yuan, err)
		}
		if back != fen {
			t.Fatalf("往返不一致: fen=%d → yuan=%v → fen=%d", fen, yuan, back)
		}
	}
}

// TestAmountFromFenFallbackWhenNil 未回填的库（amount_fen 为 NULL）必须回落旧列，
// 不能把 NULL 读成 0 元 —— 这是 AmountFen 用 *int64 而不是 int64 的唯一理由。
func TestAmountFromFenFallbackWhenNil(t *testing.T) {
	fen := int64(12345)
	rows := []struct {
		name string
		in   entity.Transaction
		want float64
	}{
		{"未回填读旧列", entity.Transaction{Amount: 88.5, AmountFen: nil}, 88.5},
		{"已回填读分", entity.Transaction{Amount: 0, AmountFen: &fen}, 123.45},
	}
	for _, r := range rows {
		item := r.in
		applyAmountFromFen(&item)
		if item.Amount != r.want {
			t.Errorf("%s: Amount = %v, 期望 %v", r.name, item.Amount, r.want)
		}
	}
	applyAmountFromFen(nil) // 不 panic 即可
}
