// mall_controller.go 门店商城 HTTP。
package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// MallController 商城。
type MallController struct {
	mallUC *usecase.MallUsecase
}

// NewMallController 创建。
func NewMallController(mallUC *usecase.MallUsecase) *MallController {
	return &MallController{mallUC: mallUC}
}

// CreateProduct POST /api/v1/mall/products
func (ctrl *MallController) CreateProduct(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	p, err := ctrl.mallUC.CreateProduct(c.Request.Context(), user.ID, body.StoreID, body.Kind, body.Title, body.CoverURL, body.Status)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.SuccessCreated(c, p)
}

// CreateSKU POST /api/v1/mall/products/:product_id/skus
func (ctrl *MallController) CreateSKU(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
		StockQty    int             `json:"stock_qty"`
		Status      int16           `json:"status"`
		DeliverType *int16          `json:"deliver_type"`
		ContentURL  *string         `json:"content_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	sku, err := ctrl.mallUC.CreateSKU(c.Request.Context(), user.ID, usecase.CreateSKUInput{
		ProductID:   productID,
		SKUCode:     body.SKUCode,
		Title:       body.Title,
		SpecsJSON:   body.Specs,
		Price:       body.Price,
		StockQty:    body.StockQty,
		Status:      body.Status,
		DeliverType: body.DeliverType,
		ContentURL:  body.ContentURL,
	})
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.SuccessCreated(c, sku)
}

// AddVirtualCodes POST /api/v1/mall/skus/:sku_id/codes
func (ctrl *MallController) AddVirtualCodes(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	if err := ctrl.mallUC.AddVirtualCodes(c.Request.Context(), user.ID, skuID, body.Codes); err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// GetProduct GET /api/v1/mall/stores/:store_id/products/:product_id
func (ctrl *MallController) GetProduct(c *gin.Context) {
	if _, _, ok := supabaseAuthContext(c); !ok {
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
	detail, err := ctrl.mallUC.GetShelfProduct(c.Request.Context(), storeID, productID)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, detail)
}

// ListProducts GET /api/v1/mall/stores/:store_id/products?page=&size=
func (ctrl *MallController) ListProducts(c *gin.Context) {
	if _, _, ok := supabaseAuthContext(c); !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := parsePathStoreID(c.Param("store_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := ctrl.mallUC.ListShelfItems(c.Request.Context(), storeID, pq.Page, pq.Size)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

// UpsertCart POST /api/v1/mall/cart
func (ctrl *MallController) UpsertCart(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	item, err := ctrl.mallUC.UpsertCart(c.Request.Context(), user.ID, body.SKUID, body.Qty)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, item)
}

// ListCart GET /api/v1/mall/cart
func (ctrl *MallController) ListCart(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	items, err := ctrl.mallUC.ListCart(c.Request.Context(), user.ID)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, gin.H{"items": items})
}

// CreateOrder POST /api/v1/mall/orders
func (ctrl *MallController) CreateOrder(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	lines := make([]usecase.CreateOrderLine, 0, len(body.Lines))
	for _, l := range body.Lines {
		lines = append(lines, usecase.CreateOrderLine{SKUID: l.SKUID, Qty: l.Qty})
	}
	detail, err := ctrl.mallUC.CreateOrder(c.Request.Context(), user.ID, usecase.CreateOrderInput{
		IdempotencyKey:  body.IdempotencyKey,
		StoreID:         body.StoreID,
		Lines:           lines,
		ReceiverName:    body.ReceiverName,
		ReceiverPhone:   body.ReceiverPhone,
		ReceiverAddress: body.ReceiverAddress,
		ClearCart:       body.ClearCart,
	})
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.SuccessCreated(c, detail)
}

// GetOrder GET /api/v1/mall/orders/:order_id
func (ctrl *MallController) GetOrder(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "order_id 无效")
		return
	}
	detail, err := ctrl.mallUC.GetOrder(c.Request.Context(), user.ID, orderID)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, detail)
}

// PayOrder POST /api/v1/mall/orders/:order_id/pay
func (ctrl *MallController) PayOrder(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	detail, err := ctrl.mallUC.PayOrder(c.Request.Context(), user.ID, orderID, body.PaymentChannel)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, detail)
}

// CancelOrder POST /api/v1/mall/orders/:order_id/cancel
func (ctrl *MallController) CancelOrder(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	orderID, err := strconv.ParseInt(c.Param("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "order_id 无效")
		return
	}
	order, err := ctrl.mallUC.CancelOrder(c.Request.Context(), user.ID, orderID)
	if err != nil {
		writeMallError(c, err)
		return
	}
	response.Success(c, order)
}

func writeMallError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMallForbidden), errors.Is(err, usecase.ErrAccessForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, err.Error())
	case errors.Is(err, usecase.ErrMallNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	case errors.Is(err, usecase.ErrMallInvalidChannel),
		errors.Is(err, usecase.ErrMallInvalidKind),
		errors.Is(err, usecase.ErrMallInvalidPrice),
		errors.Is(err, usecase.ErrMallInvalidQty),
		errors.Is(err, usecase.ErrMallNeedAddress),
		errors.Is(err, usecase.ErrMallMultiStore),
		errors.Is(err, usecase.ErrMallEmptyCart),
		errors.Is(err, usecase.ErrMallInvalidIdempotency),
		errors.Is(err, usecase.ErrMallInvalidTitle),
		errors.Is(err, usecase.ErrMallSKUOffShelf),
		errors.Is(err, usecase.ErrMallStockInsufficient),
		errors.Is(err, usecase.ErrMallCodeInsufficient),
		errors.Is(err, usecase.ErrMallOrderNotPayable),
		errors.Is(err, usecase.ErrMallOrderNotCancelable):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	default:
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	}
}
