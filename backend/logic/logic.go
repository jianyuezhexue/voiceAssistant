package logic

import (
	"github.com/gin-gonic/gin"
)

// BaseLogic 基础逻辑层
type BaseLogic struct {
	Ctx *gin.Context
}

// NewBaseLogic 创建基础逻辑层
func NewBaseLogic(ctx *gin.Context) BaseLogic {
	return BaseLogic{Ctx: ctx}
}
