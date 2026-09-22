package handler

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/blueship581/codelearn/internal/constants"
	"github.com/blueship581/codelearn/internal/service"
	"github.com/blueship581/codelearn/internal/util"
)

// RejudgeHandler 提交重判处理器（仅管理员可发起；历史本人/管理员可查）。
type RejudgeHandler struct {
	rejudgeService *service.RejudgeService
}

// NewRejudgeHandler 构造重判处理器。
func NewRejudgeHandler(rejudgeService *service.RejudgeService) *RejudgeHandler {
	return &RejudgeHandler{rejudgeService: rejudgeService}
}

// Rejudge 发起重判（管理员）。
func (h *RejudgeHandler) Rejudge(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		_ = c.Error(util.NewAppError(constants.CodeBadRequest, "无效的提交 ID"))
		return
	}
	resp, err := h.rejudgeService.Rejudge(c.Request.Context(), id, getUserID(c), getUsername(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, resp)
}

// History 查询某条提交的历次重判记录（本人或管理员）。
func (h *RejudgeHandler) History(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		_ = c.Error(util.NewAppError(constants.CodeBadRequest, "无效的提交 ID"))
		return
	}
	list, err := h.rejudgeService.ListHistory(c.Request.Context(), id, getUserID(c), getRole(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	util.Success(c, list)
}
