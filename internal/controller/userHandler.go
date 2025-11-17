package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/middleware"
)

func AuthMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy user từ context (đã được set bởi middleware)
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found in context"})
			return
		}

		// Convert user từ context về UserContext type
		userInfo, ok := user.(middleware.UserContext)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Invalid user data"})
			return
		}

		// Trả về thông tin user
		c.JSON(http.StatusOK, gin.H{
			"user": userInfo,
		})
	}
}