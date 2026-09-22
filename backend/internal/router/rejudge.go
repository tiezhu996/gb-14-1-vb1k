package router

import (
	"github.com/gin-gonic/gin"

	"github.com/blueship581/codelearn/internal/handler"
)

// registerRejudgeRoutes 注册提交重判路由。
// POST 为管理员写操作（经 admin + audit 中间件）；GET 历史本人/管理员可读。
func registerRejudgeRoutes(api *gin.RouterGroup, h *handler.RejudgeHandler, auth, admin, audit gin.HandlerFunc) {
	api.POST("/submissions/:id/rejudge", auth, admin, audit, h.Rejudge)
	api.GET("/submissions/:id/rejudges", auth, h.History)
}
