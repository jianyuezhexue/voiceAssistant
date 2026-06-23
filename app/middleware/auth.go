package middleware

import "github.com/gin-gonic/gin"

// AuthMiddleware 设置默认用户上下文
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// todo 增加鉴权的逻辑

		c.Set("currUserId", "1")
		c.Set("currUserName", "系统")
		c.Next()
	}
}
