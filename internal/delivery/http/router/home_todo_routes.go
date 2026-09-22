package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerHomeTodoRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.HomeTodoController, accountGate gin.HandlerFunc) {
	if ctrl == nil {
		return
	}
	chain := sessionAuthChain(sbAuth, accountGate)

	home := v1.Group("/home")
	home.Use(chain...)
	{
		home.GET("/todo-cards", ctrl.ListTodoCards)
		home.GET("/join-applications", ctrl.ListPendingJoinApplications)
		home.POST("/join-applications/:application_id/approve", ctrl.ApproveJoinApplication)
		home.POST("/join-applications/:application_id/reject", ctrl.RejectJoinApplication)
		home.GET("/follow-up-customers", ctrl.ListOverdueCustomers)
		home.GET("/after-sales-appointments", ctrl.ListPendingAppointments)
		home.GET("/store-review-orders", ctrl.ListPendingReviewOrders)
	}

	stores := v1.Group("/stores")
	stores.Use(chain...)
	{
		stores.POST("/:store_id/join-applications", ctrl.ApplyToStore)
	}
}
