package web

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hex-techs/blade/pkg/utils/token"
)

const (
	CurrentUser   = "user"
	defaultBearer = "Bearer"
)

func GetCurrentUser(c *gin.Context) *token.Claims {
	u := c.MustGet(CurrentUser).(*token.Claims)
	if u == nil {
		return &token.Claims{}
	}
	return u
}

// 登录验证中间件
func LoginRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从header中获取token
		t := c.Request.Header.Get("Authorization")
		if t == "" {
			c.JSON(http.StatusUnauthorized, ExceptResponse(http.StatusUnauthorized, "Unauthorized"))
			c.Abort()
			return
		}
		tarr := strings.Split(t, " ")
		if len(tarr) != 2 || tarr[0] != defaultBearer {
			c.JSON(http.StatusUnauthorized, ExceptResponse(http.StatusUnauthorized, "Unauthorized"))
			c.Abort()
			return
		}
		// 解析token
		claims, err := token.ParseJWTToken(tarr[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, ExceptResponse(http.StatusUnauthorized, err))
			c.Abort()
			return
		}
		// 将用户信息保存到上下文中
		c.Set("user", claims)
		c.Next()
	}
}

// 必须是管理员才能访问的中间件
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetCurrentUser(c)
		if !claims.Admin {
			c.JSON(http.StatusForbidden, ExceptResponse(http.StatusForbidden, "Forbidden"))
			c.Abort()
			return
		}
		c.Next()
	}
}
