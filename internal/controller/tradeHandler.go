package controller

import (
	"net/http"
	"strconv"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"database/sql"
)

func GetUserTradesHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user context from middleware
		userInterface, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Type assert to UserContext struct
		user, ok := userInterface.(middleware.UserContext)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}

		// Get user ID as string
		userId := user.ID.String()

		// Get limit from query params (default 100)
		limitStr := c.DefaultQuery("limit", "100")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 100
		}

		// Fetch trades
		tradeRepo := repo.NewTradeRepo(db)
		trades, err := tradeRepo.GetByUserId(c.Request.Context(), userId, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trades"})
			return
		}

		c.JSON(http.StatusOK, trades)
	}
}

