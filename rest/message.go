package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ismdeep/log"
	"go.uber.org/zap"

	"github.com/ismdeep/notification-gateway/core/input"
)

func (r *Rest) PushMessage(ctx *gin.Context) {
	var msg input.Message
	if err := ctx.ShouldBindJSON(&msg); err != nil {
		log.WithContext(ctx).Error("bind message failed", zap.Error(err))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"msg": ErrRequestBindJSON.Error()})
		return
	}
	r.inputChan <- msg
	ctx.JSON(http.StatusOK, gin.H{"msg": "ok"})
}
