package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

// registerMallRoutes 商城。仅 local 注册。
func registerMallRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, mallCtrl *controller.MallController, accountGate gin.HandlerFunc) {
	if mallCtrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)

	mall := v1.Group("/mall")
	mall.Use(chain...)
	{
		mall.POST("/products", mallCtrl.CreateProduct)
		mall.POST("/products/:product_id/skus", mallCtrl.CreateSKU)
		mall.POST("/skus/:sku_id/codes", mallCtrl.AddVirtualCodes)

		mall.GET("/stores/:store_id/products", mallCtrl.ListProducts)
		mall.GET("/stores/:store_id/products/:product_id", mallCtrl.GetProduct)

		mall.POST("/cart", mallCtrl.UpsertCart)
		mall.GET("/cart", mallCtrl.ListCart)

		mall.GET("/orders", mallCtrl.ListOrders)
		mall.POST("/orders", mallCtrl.CreateOrder)
		mall.GET("/orders/:order_id", mallCtrl.GetOrder)
		mall.POST("/orders/:order_id/pay", mallCtrl.PayOrder)
		mall.POST("/orders/:order_id/cancel", mallCtrl.CancelOrder)
	}
}
