// handler.go 门店商城 HTTP。
package mall

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"app/internal/access"
	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"app/internal/points"
	"app/internal/wallet"
	"github.com/gin-gonic/gin"
)

// handler 门店商城 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/mall_controller.go 迁入。
// 类型改小写 handler；取身份改走 authctx.SupabaseAuthContext；
// parsePathStoreID 回到域内私有（原在 controller/params.go 的等价副本）。
type handler struct {
	svc *Service
}

// newHandler 创建。
func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

// CreateProduct POST /api/v1/mall/products
func (h *handler) CreateProduct(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		StoreID  int     `json:"store_id"`
		Kind     int16   `json:"kind"`
		Title    string  `json:"title"`
		CoverURL *string `json:"cover_url"`
		Status   int16   `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	p, err := h.svc.CreateProduct(c.Request.Context(), user.ID, body.StoreID, body.Kind, body.Title, body.CoverURL, body.Status)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, p)
}

// CreateSKU POST /api/v1/mall/products/:product_id/skus
func (h *handler) CreateSKU(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "product_id 无效")
		return
	}
	var body struct {
		SKUCode     string          `json:"sku_code"`
		Title       string          `json:"title"`
		Specs       json.RawMessage `json:"specs"`
		Price       string          `json:"price"`
		PricePoints int64           `json:"price_points"`
		StockQty    int             `json:"stock_qty"`
		Status      int16           `json:"status"`
		DeliverType *int16          `json:"deliver_type"`
		ContentURL  *string         `json:"content_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	sku, err := h.svc.CreateSKU(c.Request.Context(), user.ID, CreateSKUInput{
		ProductID:   productID,
		SKUCode:     body.SKUCode,
		Title:       body.Title,
		SpecsJSON:   body.Specs,
		Price:       body.Price,
		PricePoints: body.PricePoints,
		StockQty:    body.StockQty,
		Status:      body.Status,
		DeliverType: body.DeliverType,
		ContentURL:  body.ContentURL,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, sku)
}

// AddVirtualCodes POST /api/v1/mall/skus/:sku_id/codes
func (h *handler) AddVirtualCodes(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	skuID, err := strconv.ParseInt(c.Param("sku_id"), 10, 64)
	if err != nil || skuID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "sku_id 无效")
		return
	}
	var body struct {
		Codes []string `json:"codes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := h.svc.AddVirtualCodes(c.Request.Context(), user.ID, skuID, body.Codes); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// GetProduct GET /api/v1/mall/stores/:store_id/products/:product_id
func (h *handler) GetProduct(c *gin.Context) {
	if _, _, ok := authctx.SupabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := parsePathStoreID(c.Param("store_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "product_id 无效")
		return
	}
	detail, err := h.svc.GetShelfProduct(c.Request.Context(), storeID, productID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, detail)
}

// ListProducts GET /api/v1/mall/stores/:store_id/products?page=&size=
func (h *handler) ListProducts(c *gin.Context) {
	if _, _, ok := authctx.SupabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := parsePathStoreID(c.Param("store_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := h.svc.ListShelfItems(c.Request.Context(), storeID, pq.Page, pq.Size)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

// UpsertCart POST /api/v1/mall/cart
func (h *handler) UpsertCart(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		SKUID int64 `json:"sku_id"`
		Qty   int   `json:"qty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	item, err := h.svc.UpsertCart(c.Request.Context(), user.ID, body.SKUID, body.Qty)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, item)
}

// ListCart GET /api/v1/mall/cart
func (h *handler) ListCart(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	items, err := h.svc.ListCart(c.Request.Context(), user.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

// ListOrders GET /api/v1/mall/orders?page=&size=&status=
// status 可省略（全部），或单个 0–4，或逗号分隔多值（如 1,2）。
func (h *handler) ListOrders(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	statuses, err := parseOrderStatusQuery(c.Query("status"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := h.svc.ListOrders(c.Request.Context(), user.ID, statuses, pq.Page, pq.Size)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

// CreateOrder POST /api/v1/mall/orders
func (h *handler) CreateOrder(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		IdempotencyKey  string `json:"idempotency_key"`
		StoreID         int    `json:"store_id"`
		ClearCart       bool   `json:"clear_cart"`
		ReceiverName    string `json:"receiver_name"`
		ReceiverPhone   string `json:"receiver_phone"`
		ReceiverAddress string `json:"receiver_address"`
		Lines           []struct {
			SKUID int64 `json:"sku_id"`
			Qty   int   `json:"qty"`
		} `json:"lines"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	lines := make([]CreateOrderLine, 0, len(body.Lines))
	for _, l := range body.Lines {
		lines = append(lines, CreateOrderLine{SKUID: l.SKUID, Qty: l.Qty})
	}
	detail, err := h.svc.CreateOrder(c.Request.Context(), user.ID, CreateOrderInput{
		IdempotencyKey:  body.IdempotencyKey,
		StoreID:         body.StoreID,
		Lines:           lines,
		ReceiverName:    body.ReceiverName,
		ReceiverPhone:   body.ReceiverPhone,
		ReceiverAddress: body.ReceiverAddress,
		ClearCart:       body.ClearCart,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, detail)
}

// GetOrder GET /api/v1/mall/orders/:order_id
func (h *handler) GetOrder(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "order_id 无效")
		return
	}
	detail, err := h.svc.GetOrder(c.Request.Context(), user.ID, orderID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, detail)
}

// PayOrder POST /api/v1/mall/orders/:order_id/pay
func (h *handler) PayOrder(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "order_id 无效")
		return
	}
	var body struct {
		PaymentChannel int16 `json:"payment_channel"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	res, err := h.svc.PayOrder(c.Request.Context(), user.ID, orderID, body.PaymentChannel)
	if err != nil {
		writeError(c, err)
		return
	}
	// 渠道支付：回 prepay 参数，订单仍是待支付（回调核销后才变已支付）。
	if res.Status == "pending" {
		response.Success(c, res)
		return
	}
	response.Success(c, res.Detail)
}

// CancelOrder POST /api/v1/mall/orders/:order_id/cancel
func (h *handler) CancelOrder(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "order_id 无效")
		return
	}
	order, err := h.svc.CancelOrder(c.Request.Context(), user.ID, orderID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, order)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden), errors.Is(err, access.ErrForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, err.Error())
	case errors.Is(err, ErrNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	case errors.Is(err, ErrOrderAlreadySettled):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, err.Error())
	case errors.Is(err, ErrPayGatewayUnavailable):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, ErrInvalidChannel),
		errors.Is(err, ErrInvalidKind),
		errors.Is(err, ErrInvalidPrice),
		errors.Is(err, ErrInvalidQty),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrNeedAddress),
		errors.Is(err, ErrMultiStore),
		errors.Is(err, ErrEmptyCart),
		errors.Is(err, ErrInvalidIdempotency),
		errors.Is(err, ErrInvalidTitle),
		errors.Is(err, ErrSKUOffShelf),
		errors.Is(err, ErrStockInsufficient),
		errors.Is(err, ErrCodeInsufficient),
		errors.Is(err, ErrOrderNotPayable),
		errors.Is(err, ErrOrderPayExpired),
		errors.Is(err, ErrOrderNotCancelable),
		errors.Is(err, ErrPaymentModeMixed),
		errors.Is(err, ErrNeedCNYChannel),
		errors.Is(err, points.ErrPointsInsufficient),
		errors.Is(err, wallet.ErrCashInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	default:
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	}
}

func parseOrderStatusQuery(raw string) ([]int16, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int16, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseInt(p, 10, 16)
		if err != nil || v < 0 || v > 4 {
			return nil, ErrInvalidStatus
		}
		out = append(out, int16(v))
	}
	if len(out) == 0 {
		return nil, ErrInvalidStatus
	}
	return out, nil
}

// parsePathStoreID 解析路径里的 store_id；非法时返回 access.ErrInvalidStoreID。
//
// 原在 delivery/http/controller/params.go（因 mall controller 当时留在 controller
// 包而留的等价副本，access 域的私有实现随批 1 迁走）。本域切片后回到域内私有，
// controller/params.go 随之删除。
func parsePathStoreID(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, access.ErrInvalidStoreID
	}
	return n, nil
}
