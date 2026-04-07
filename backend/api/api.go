package api

import (
	"github.com/gin-gonic/gin"
)

// Base API 基础控制器
type Base struct {
	Ctx *gin.Context
}

// Bind 绑定请求参数
func (a *Base) Bind(ctx *gin.Context, req interface{}) error {
	a.Ctx = ctx // 重要：必须设置 Ctx，否则 Error 和 Success 方法会 panic
	if err := ctx.ShouldBind(req); err != nil {
		return err
	}
	return nil
}

// Success 成功响应
func (a *Base) Success(data interface{}, message string) {
	a.Ctx.JSON(200, gin.H{
		"code":    0,
		"message": message,
		"data":    data,
	})
}

// Error 错误响应
func (a *Base) Error(err error) {
	a.Ctx.JSON(200, gin.H{
		"code":    -1,
		"message": err.Error(),
	})
}
