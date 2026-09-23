package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MallRepository 商城 Postgres 实现。
type MallRepository struct {
	db *gorm.DB
}

// NewMallRepository 创建。
func NewMallRepository(db *gorm.DB) *MallRepository {
	return &MallRepository{db: db}
}

func (r *MallRepository) CreateProduct(ctx context.Context, p *entity.WysMallProduct) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *MallRepository) CreateSKU(ctx context.Context, sku *entity.WysMallSKU) error {
	if len(sku.Specs) == 0 {
		sku.Specs = json.RawMessage(`{}`)
	}
	return r.db.WithContext(ctx).Create(sku).Error
}

func (r *MallRepository) AddVirtualCodes(ctx context.Context, skuID int64, codes []string) error {
	rows := make([]entity.WysMallVirtualCode, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		rows = append(rows, entity.WysMallVirtualCode{
			SKUID:  skuID,
			Code:   c,
			Status: entity.MallCodeUnused,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *MallRepository) GetProduct(ctx context.Context, productID int64) (*entity.WysMallProduct, error) {
	var p entity.WysMallProduct
	err := r.db.WithContext(ctx).Where("product_id = ? AND deleted_at IS NULL", productID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrMallNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *MallRepository) ListOnShelfSKUs(ctx context.Context, productID int64) ([]entity.WysMallSKU, error) {
	var skus []entity.WysMallSKU
	err := r.db.WithContext(ctx).
		Where("product_id = ? AND status = ? AND deleted_at IS NULL", productID, entity.MallSKUOn).
		Order("sku_id").
		Find(&skus).Error
	return skus, err
}

func (r *MallRepository) GetSKU(ctx context.Context, skuID int64) (*entity.WysMallSKU, error) {
	var s entity.WysMallSKU
	err := r.db.WithContext(ctx).Where("sku_id = ? AND deleted_at IS NULL", skuID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrMallNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MallRepository) ListOnShelfProducts(ctx context.Context, storeID int) ([]entity.MallProductWithSKUs, error) {
	items, _, err := r.ListShelfItems(ctx, storeID, 0, 1000)
	if err != nil {
		return nil, err
	}
	out := make([]entity.MallProductWithSKUs, 0, len(items))
	for _, it := range items {
		cover := it.CoverURL
		p := entity.WysMallProduct{
			ProductID:   it.ProductID,
			StoreID:     storeID,
			Kind:        it.Kind,
			Title:       it.Title,
			CoverURL:    &cover,
			CoverAspect: it.CoverAspect,
			Status:      entity.MallProductOnShelf,
		}
		sku := entity.WysMallSKU{
			SKUID:     it.SKUID,
			ProductID: it.ProductID,
			Title:     it.Subtitle,
			Price:     it.Price,
			Status:    entity.MallSKUOn,
		}
		out = append(out, entity.MallProductWithSKUs{Product: p, SKUs: []entity.WysMallSKU{sku}})
	}
	return out, nil
}

func (r *MallRepository) ListShelfItems(ctx context.Context, storeID, offset, limit int) ([]entity.MallShelfItem, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	countSQL := `
		SELECT COUNT(*)
		FROM wys_mall_product p
		WHERE p.store_id = ?
		  AND p.status = ?
		  AND p.deleted_at IS NULL
		  AND EXISTS (
		    SELECT 1 FROM wys_mall_sku s
		    WHERE s.product_id = p.product_id AND s.status = ? AND s.deleted_at IS NULL
		  )`
	if err := r.db.WithContext(ctx).Raw(countSQL, storeID, entity.MallProductOnShelf, entity.MallSKUOn).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	type row struct {
		ProductID   int64   `gorm:"column:product_id"`
		SKUID       int64   `gorm:"column:sku_id"`
		Title       string  `gorm:"column:title"`
		CoverURL    *string `gorm:"column:cover_url"`
		CoverAspect float64 `gorm:"column:cover_aspect"`
		Kind        int16   `gorm:"column:kind"`
		Price       string  `gorm:"column:price"`
		PricePoints int64   `gorm:"column:price_points"`
		SKUTitle    string  `gorm:"column:sku_title"`
	}
	var rows []row
	listSQL := `
		SELECT p.product_id, s.sku_id, p.title, p.cover_url, p.cover_aspect, p.kind,
		       s.price::text AS price, COALESCE(s.price_points, 0) AS price_points, s.title AS sku_title
		FROM wys_mall_product p
		INNER JOIN LATERAL (
		  SELECT sku_id, price, price_points, title
		  FROM wys_mall_sku
		  WHERE product_id = p.product_id AND status = ? AND deleted_at IS NULL
		  ORDER BY sku_id
		  LIMIT 1
		) s ON true
		WHERE p.store_id = ? AND p.status = ? AND p.deleted_at IS NULL
		ORDER BY p.product_id
		LIMIT ? OFFSET ?`
	if err := r.db.WithContext(ctx).Raw(
		listSQL,
		entity.MallSKUOn, storeID, entity.MallProductOnShelf, limit, offset,
	).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]entity.MallShelfItem, 0, len(rows))
	for _, rw := range rows {
		cover := ""
		if rw.CoverURL != nil {
			cover = *rw.CoverURL
		}
		aspect := rw.CoverAspect
		if aspect < 0.5 {
			aspect = 1
		}
		subtitle := rw.SKUTitle
		if subtitle == "" {
			if rw.Kind == entity.MallKindVirtual {
				subtitle = "虚拟商品"
			} else {
				subtitle = "实体发货"
			}
		}
		out = append(out, entity.MallShelfItem{
			ProductID:   rw.ProductID,
			SKUID:       rw.SKUID,
			Title:       rw.Title,
			CoverURL:    cover,
			CoverAspect: aspect,
			Kind:        rw.Kind,
			Price:       rw.Price,
			PricePoints: rw.PricePoints,
			PaymentMode: entity.MallSKUPaymentMode(rw.Price, rw.PricePoints),
			Subtitle:    subtitle,
		})
	}
	return out, total, nil
}

func (r *MallRepository) UpsertCartItem(ctx context.Context, item *entity.WysMallCartItem) error {
	item.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "sku_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"qty", "updated_at"}),
	}).Create(item).Error
}

func (r *MallRepository) ListCart(ctx context.Context, userID string) ([]entity.WysMallCartItem, error) {
	var items []entity.WysMallCartItem
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("sku_id").Find(&items).Error
	return items, err
}

func (r *MallRepository) ClearCartSKUs(ctx context.Context, userID string, skuIDs []int64) error {
	if len(skuIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("user_id = ? AND sku_id IN ?", userID, skuIDs).Delete(&entity.WysMallCartItem{}).Error
}

func (r *MallRepository) FindOrderByIdempotency(ctx context.Context, buyerUserID, key string) (*entity.WysMallOrder, error) {
	var o entity.WysMallOrder
	err := r.db.WithContext(ctx).Where("buyer_user_id = ? AND idempotency_key = ?", buyerUserID, key).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *MallRepository) CreateOrder(ctx context.Context, order *entity.WysMallOrder, items []entity.WysMallOrderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].OrderID = order.OrderID
			if len(items[i].Specs) == 0 {
				items[i].Specs = json.RawMessage(`{}`)
			}
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		from := (*int16)(nil)
		log := entity.WysMallOrderLog{
			OrderID:     order.OrderID,
			FromStatus:  from,
			ToStatus:    entity.MallOrderUnpaid,
			ActorUserID: order.BuyerUserID,
		}
		return tx.Create(&log).Error
	})
}

func (r *MallRepository) GetOrder(ctx context.Context, orderID int64) (*entity.WysMallOrder, error) {
	var o entity.WysMallOrder
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrMallNotFound
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *MallRepository) ListOrdersByBuyer(ctx context.Context, buyerUserID string, statuses []int16, offset, limit int) ([]entity.WysMallOrder, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.WysMallOrder{}).Where("buyer_user_id = ?", buyerUserID)
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []entity.WysMallOrder
	err := q.Order("created_at DESC, order_id DESC").Offset(offset).Limit(limit).Find(&orders).Error
	return orders, total, err
}

func (r *MallRepository) ListOrderItems(ctx context.Context, orderID int64) ([]entity.WysMallOrderItem, error) {
	var items []entity.WysMallOrderItem
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("item_id").Find(&items).Error
	return items, err
}

func (r *MallRepository) ListOrderItemsByOrderIDs(ctx context.Context, orderIDs []int64) ([]entity.WysMallOrderItem, error) {
	if len(orderIDs) == 0 {
		return nil, nil
	}
	var items []entity.WysMallOrderItem
	err := r.db.WithContext(ctx).Where("order_id IN ?", orderIDs).Order("order_id, item_id").Find(&items).Error
	return items, err
}

func (r *MallRepository) ListPayments(ctx context.Context, orderID int64) ([]entity.WysMallPayment, error) {
	var rows []entity.WysMallPayment
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("payment_id").Find(&rows).Error
	return rows, err
}

func (r *MallRepository) PayOrderLocal(ctx context.Context, orderID int64, buyerUserID string, channel int16) (*entity.MallOrderDetail, error) {
	var detail entity.MallOrderDetail
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order entity.WysMallOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_id = ? AND buyer_user_id = ?", orderID, buyerUserID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrMallNotFound
			}
			return err
		}
		if order.Status == entity.MallOrderPaid || order.Status == entity.MallOrderFulfilled {
			items, err := listOrderItemsTx(tx, orderID)
			if err != nil {
				return err
			}
			pays, err := listPaymentsTx(tx, orderID)
			if err != nil {
				return err
			}
			detail = entity.MallOrderDetail{Order: order, Items: items, Payments: pays}
			return nil
		}
		if order.Status != entity.MallOrderUnpaid {
			return repository.ErrMallOrderNotPayable
		}

		items, err := listOrderItemsTx(tx, orderID)
		if err != nil {
			return err
		}

		now := time.Now()
		for _, it := range items {
			if it.Kind == entity.MallKindPhysical {
				res := tx.Exec(`
					UPDATE wys_mall_sku
					SET stock_qty = stock_qty - ?, updated_at = now()
					WHERE sku_id = ? AND deleted_at IS NULL AND stock_qty >= ?
				`, it.Qty, it.SKUID, it.Qty)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					return repository.ErrMallStockInsufficient
				}
			} else if it.Kind == entity.MallKindVirtual {
				var sku entity.WysMallSKU
				if err := tx.Where("sku_id = ? AND deleted_at IS NULL", it.SKUID).First(&sku).Error; err != nil {
					return err
				}
				if sku.DeliverType == nil {
					return repository.ErrMallInvalidKind
				}
				switch *sku.DeliverType {
				case entity.MallDeliverContentURL:
					if sku.ContentURL == nil || strings.TrimSpace(*sku.ContentURL) == "" {
						return repository.ErrMallInvalidKind
					}
					url := *sku.ContentURL
					if err := tx.Model(&entity.WysMallOrderItem{}).
						Where("item_id = ?", it.ItemID).
						Update("content_url", url).Error; err != nil {
						return err
					}
				case entity.MallDeliverRedeemCode:
					var codes []entity.WysMallVirtualCode
					if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
						Where("sku_id = ? AND status = ?", it.SKUID, entity.MallCodeUnused).
						Order("code_id").
						Limit(it.Qty).
						Find(&codes).Error; err != nil {
						return err
					}
					if len(codes) < it.Qty {
						return repository.ErrMallCodeInsufficient
					}
					for _, c := range codes {
						itemID := it.ItemID
						if err := tx.Model(&entity.WysMallVirtualCode{}).
							Where("code_id = ? AND status = ?", c.CodeID, entity.MallCodeUnused).
							Updates(map[string]interface{}{
								"status":        entity.MallCodeIssued,
								"order_item_id": itemID,
							}).Error; err != nil {
							return err
						}
					}
				default:
					return repository.ErrMallInvalidKind
				}
			}
		}

		tradeNo := "local-" + uuid.NewString()

		// 积分支付单（余额已由 PointsUsecase 在 usecase 层扣减）
		if order.TotalPoints > 0 {
			ptsTrade := "pts-" + uuid.NewString()
			ptsPay := entity.WysMallPayment{
				PaymentNo:      "P" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20],
				OrderID:        orderID,
				UserID:         buyerUserID,
				PaymentChannel: entity.MallPayPoints,
				Amount:         "0.00",
				Status:         entity.MallPaymentSuccess,
				ChannelTradeNo: &ptsTrade,
				ChannelPayload: json.RawMessage(fmt.Sprintf(`{"points":%d}`, order.TotalPoints)),
				PaidAt:         &now,
			}
			if err := tx.Create(&ptsPay).Error; err != nil {
				return err
			}
		}

		payChannel := channel
		if entity.IsZeroMoney(order.Amount) {
			payChannel = entity.MallPayPoints
		} else {
			cnyTrade := tradeNo
			pay := entity.WysMallPayment{
				PaymentNo:      "P" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20],
				OrderID:        orderID,
				UserID:         buyerUserID,
				PaymentChannel: channel,
				Amount:         order.Amount,
				Status:         entity.MallPaymentSuccess,
				ChannelTradeNo: &cnyTrade,
				ChannelPayload: json.RawMessage(`{}`),
				PaidAt:         &now,
			}
			if err := tx.Create(&pay).Error; err != nil {
				return err
			}
			payChannel = channel
		}

		from := entity.MallOrderUnpaid
		res := tx.Model(&entity.WysMallOrder{}).
			Where("order_id = ? AND status = ?", orderID, entity.MallOrderUnpaid).
			Updates(map[string]interface{}{
				"status":          entity.MallOrderPaid,
				"payment_channel": payChannel,
				"paid_at":         now,
				"updated_at":      now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return repository.ErrMallOrderNotPayable
		}

		logRow := entity.WysMallOrderLog{
			OrderID:     orderID,
			FromStatus:  &from,
			ToStatus:    entity.MallOrderPaid,
			ActorUserID: buyerUserID,
		}
		if err := tx.Create(&logRow).Error; err != nil {
			return err
		}

		detailJSON, _ := json.Marshal(map[string]interface{}{
			"order_no":        order.OrderNo,
			"payment_channel": payChannel,
			"total_points":    order.TotalPoints,
		})
		audit := entity.WysMallAuditLog{
			ActorUserID: buyerUserID,
			Action:      "payment.success",
			TargetType:  "order",
			TargetID:    fmt.Sprintf("%d", orderID),
			Detail:      detailJSON,
		}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}

		updated, err := getOrderTx(tx, orderID)
		if err != nil {
			return err
		}
		items, err = listOrderItemsTx(tx, orderID)
		if err != nil {
			return err
		}
		pays, err := listPaymentsTx(tx, orderID)
		if err != nil {
			return err
		}
		detail = entity.MallOrderDetail{Order: *updated, Items: items, Payments: pays}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func (r *MallRepository) CancelUnpaidOrder(ctx context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error) {
	var out entity.WysMallOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order entity.WysMallOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_id = ? AND buyer_user_id = ?", orderID, buyerUserID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrMallNotFound
			}
			return err
		}
		if order.Status != entity.MallOrderUnpaid {
			return repository.ErrMallOrderNotCancelable
		}
		from := entity.MallOrderUnpaid
		now := time.Now()
		if err := tx.Model(&entity.WysMallOrder{}).
			Where("order_id = ? AND status = ?", orderID, entity.MallOrderUnpaid).
			Updates(map[string]interface{}{
				"status":     entity.MallOrderCancelled,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		logRow := entity.WysMallOrderLog{
			OrderID:     orderID,
			FromStatus:  &from,
			ToStatus:    entity.MallOrderCancelled,
			ActorUserID: buyerUserID,
		}
		if err := tx.Create(&logRow).Error; err != nil {
			return err
		}
		updated, err := getOrderTx(tx, orderID)
		if err != nil {
			return err
		}
		out = *updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *MallRepository) CancelPaidOrder(ctx context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error) {
	var out entity.WysMallOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order entity.WysMallOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_id = ? AND buyer_user_id = ?", orderID, buyerUserID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrMallNotFound
			}
			return err
		}
		if order.Status != entity.MallOrderPaid {
			return repository.ErrMallOrderNotCancelable
		}
		from := entity.MallOrderPaid
		now := time.Now()
		if err := tx.Model(&entity.WysMallOrder{}).
			Where("order_id = ? AND status = ?", orderID, entity.MallOrderPaid).
			Updates(map[string]interface{}{
				"status":     entity.MallOrderCancelled,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		logRow := entity.WysMallOrderLog{
			OrderID:     orderID,
			FromStatus:  &from,
			ToStatus:    entity.MallOrderCancelled,
			ActorUserID: buyerUserID,
		}
		if err := tx.Create(&logRow).Error; err != nil {
			return err
		}
		updated, err := getOrderTx(tx, orderID)
		if err != nil {
			return err
		}
		out = *updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func getOrderTx(tx *gorm.DB, orderID int64) (*entity.WysMallOrder, error) {
	var o entity.WysMallOrder
	if err := tx.Where("order_id = ?", orderID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func listOrderItemsTx(tx *gorm.DB, orderID int64) ([]entity.WysMallOrderItem, error) {
	var items []entity.WysMallOrderItem
	err := tx.Where("order_id = ?", orderID).Order("item_id").Find(&items).Error
	return items, err
}

func listPaymentsTx(tx *gorm.DB, orderID int64) ([]entity.WysMallPayment, error) {
	var rows []entity.WysMallPayment
	err := tx.Where("order_id = ?", orderID).Order("payment_id").Find(&rows).Error
	return rows, err
}
