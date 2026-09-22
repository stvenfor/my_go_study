package postgres

import (
	"fmt"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"gorm.io/gorm"
)

type mallSeedSpec struct {
	Title  string
	Kind   int16
	Price  string
	Aspect float64
	Stock  int
}

// EnsureMallSeed 门店 1 至少 35 条在售商品（可重复执行）。
func EnsureMallSeed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&entity.WysMallProduct{}).
		Where("store_id = ? AND deleted_at IS NULL", 1).
		Count(&count).Error; err != nil {
		return fmt.Errorf("统计商城商品失败: %w", err)
	}
	if count >= 35 {
		return ensurePointsRedeemSeed(db)
	}

	need := 35 - int(count)
	specs := mallSeedCatalog()
	for i := 0; i < need; i++ {
		sp := specs[i%len(specs)]
		n := int(count) + i + 1
		cover := fmt.Sprintf("https://picsum.photos/seed/mall_seed_%03d/400/%d", n, int(400*sp.Aspect))
		title := sp.Title
		if int(count) > 0 {
			title = fmt.Sprintf("%s #%d", sp.Title, n)
		}
		p := entity.WysMallProduct{
			StoreID:     1,
			Kind:        sp.Kind,
			Title:       title,
			CoverURL:    &cover,
			CoverAspect: sp.Aspect,
			Status:      entity.MallProductOnShelf,
		}
		if err := db.Create(&p).Error; err != nil {
			return fmt.Errorf("写入商品 %d 失败: %w", n, err)
		}
		sku := entity.WysMallSKU{
			ProductID: p.ProductID,
			SKUCode:   fmt.Sprintf("SEED-%03d", n),
			Title:     defaultSKUTitle(sp.Kind),
			Specs:     []byte(`{}`),
			Price:     sp.Price,
			StockQty:  sp.Stock,
			Status:    entity.MallSKUOn,
		}
		if sp.Kind == entity.MallKindVirtual {
			dt := entity.MallDeliverContentURL
			url := fmt.Sprintf("https://example.com/virtual/%03d", n)
			sku.DeliverType = &dt
			sku.ContentURL = &url
			sku.StockQty = 0
		}
		if err := db.Create(&sku).Error; err != nil {
			return fmt.Errorf("写入 SKU %d 失败: %w", n, err)
		}
	}
	return ensurePointsRedeemSeed(db)
}

// ensurePointsRedeemSeed 补纯积分 / 混合支付演示商品（可重复）。
func ensurePointsRedeemSeed(db *gorm.DB) error {
	type spec struct {
		code   string
		title  string
		cny    string
		points int64
	}
	seeds := []spec{
		{"SEED-PTS-ONLY", "签到积分兑换券", "0.00", 100},
		{"SEED-PTS-MIX", "积分加价礼包", "9.90", 50},
	}
	for _, s := range seeds {
		var n int64
		if err := db.Model(&entity.WysMallSKU{}).
			Where("sku_code = ? AND deleted_at IS NULL", s.code).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		cover := "https://picsum.photos/seed/" + s.code + "/400/400"
		p := entity.WysMallProduct{
			StoreID: 1, Kind: entity.MallKindVirtual, Title: s.title,
			CoverURL: &cover, CoverAspect: 1, Status: entity.MallProductOnShelf,
		}
		if err := db.Create(&p).Error; err != nil {
			return err
		}
		dt := entity.MallDeliverContentURL
		url := "https://example.com/points/" + s.code
		sku := entity.WysMallSKU{
			ProductID: p.ProductID, SKUCode: s.code, Title: "默认规格",
			Specs: []byte(`{}`), Price: s.cny, PricePoints: s.points,
			Status: entity.MallSKUOn, DeliverType: &dt, ContentURL: &url,
		}
		if err := db.Create(&sku).Error; err != nil {
			return err
		}
	}
	return nil
}

func defaultSKUTitle(kind int16) string {
	if kind == entity.MallKindVirtual {
		return "虚拟发放"
	}
	return "默认规格"
}

func mallSeedCatalog() []mallSeedSpec {
	// 35 条：实体与虚拟交错，封面比例错落。
	return []mallSeedSpec{
		{"店庆纪念马克杯", 0, "39.90", 1.30, 100},
		{"线上精品课兑换", 1, "99.00", 0.75, 0},
		{"品牌帆布袋", 0, "29.00", 1.20, 80},
		{"电子礼品卡 50 元", 1, "50.00", 0.90, 0},
		{"冬季保暖围巾", 0, "128.00", 1.50, 60},
		{"会员壁纸包", 1, "6.00", 0.70, 0},
		{"不锈钢保温杯", 0, "89.00", 1.10, 120},
		{"音频课合集", 1, "49.90", 1.00, 0},
		{"车载香薰套装", 0, "68.00", 1.25, 90},
		{"洗车服务券", 1, "39.00", 0.85, 0},
		{"运动速干T恤", 0, "79.00", 1.40, 70},
		{"保养手册电子版", 1, "12.00", 0.80, 0},
		{"蓝牙车载支架", 0, "59.00", 1.05, 110},
		{"线上试驾礼包", 1, "1.00", 0.95, 0},
		{"四季脚垫套装", 0, "199.00", 1.35, 40},
		{"钣喷优惠码", 1, "88.00", 0.72, 0},
		{"防晒冰袖", 0, "35.00", 1.15, 150},
		{"会员VIP月卡", 1, "30.00", 0.88, 0},
		{"钥匙扣礼盒", 0, "45.00", 1.08, 200},
		{"延保服务一年", 1, "299.00", 0.78, 0},
		{"便携洗车水枪", 0, "158.00", 1.22, 55},
		{"钣金知识课", 1, "19.90", 0.92, 0},
		{"儿童安全带垫", 0, "66.00", 1.18, 85},
		{"积分兑换券", 1, "0.00", 0.68, 0},
		{"雨刷器一对", 0, "48.00", 1.28, 130},
		{"新能源讲堂", 1, "25.00", 0.82, 0},
		{"后备箱收纳箱", 0, "119.00", 1.32, 45},
		{"保险讲解视频", 1, "9.90", 0.76, 0},
		{"车内空气净化器", 0, "269.00", 1.12, 35},
		{"二手车估价课", 1, "59.00", 0.98, 0},
		{"反光警示贴", 0, "18.00", 1.45, 220},
		{"销售话术手册", 1, "15.00", 0.74, 0},
		{"多功能急救包", 0, "98.00", 1.16, 75},
		{"VR看车体验码", 1, "0.00", 0.86, 0},
		{"品牌棒球帽", 0, "55.00", 1.00, 160},
	}
}
