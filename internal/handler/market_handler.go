package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
	"github.com/gin-gonic/gin"
)

type MarketHandler struct {
	marketRepo *repo.MarketRepo
}

func NewMarketHandler(marketRepo *repo.MarketRepo) *MarketHandler {
	return &MarketHandler{marketRepo: marketRepo}
}

// GetMarkets returns all active markets with their IDs and symbols
func (h *MarketHandler) GetMarkets(c *gin.Context) {
	markets, err := h.marketRepo.GetAllActiveMarkets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch markets"})
		return
	}

	c.JSON(http.StatusOK, markets)
}

// GetCandles returns historical candlestick data
func (h *MarketHandler) GetCandles(c *gin.Context) {
	symbol := c.Query("symbol")
	interval := c.Query("interval")
	limitStr := c.DefaultQuery("limit", "500")
	endTimeStr := c.Query("endTime")

	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}
	if interval == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interval is required"})
		return
	}

	limit := 500
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil {
		limit = 500
	}

	var endTime *time.Time
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endTime format, use RFC3339"})
			return
		}
		endTime = &t
	}

	candles, err := h.marketRepo.GetCandles(c.Request.Context(), symbol, interval, limit, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch candles"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"candles": candles})
}
