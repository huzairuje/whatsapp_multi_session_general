package routers

import (
	"whatsapp_multi_session_general/handler"

	"github.com/gin-gonic/gin"
)

type Router struct {
	Handler handler.Handler
}

func NewRoutes(handler handler.Handler) Router {
	return Router{
		Handler: handler,
	}
}

func (r Router) V1(router *gin.Engine) *gin.Engine {
	// Define routers
	router.GET("/qr", r.Handler.HandleQR)
	router.POST("/presence", r.Handler.ServeSendPresence)
	router.POST("/send", r.Handler.ServeSendText)
	router.POST("/send-bulk", r.Handler.ServeSendTextBulk)
	router.GET("/status", r.Handler.ServeStatus)
	router.POST("/check-user", r.Handler.ServeCheckUser)
	router.POST("/upload", r.Handler.NewUploadHandler)
	router.GET("/devices", r.Handler.ServeAllDevices)
	router.POST("/logout", r.Handler.Logout)

	return router
}
