package handler

import (
	"net/http"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct{ svc *service.OrderService }
func NewOrderHandler(s *service.OrderService) *OrderHandler { return &OrderHandler{svc:s} }

func (h *OrderHandler) Place(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req service.PlaceOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, trades, err := h.svc.PlaceOrder(c.Request.Context(), user.ID.String(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":   user,   // nếu muốn trả user về
		"order":  order,
		"trades": trades,
	})
}

func (h *OrderHandler) List(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get optional status filter from query param
	status := c.Query("status")

	orders, err := h.svc.ListOrders(c.Request.Context(), user.ID.String(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return array directly for frontend consistency
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) Cancel(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")
	if err := h.svc.CancelOrder(c.Request.Context(), user.ID.String(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "canceled"})
}

func (h *OrderHandler) Amend(c *gin.Context) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	id := c.Param("id")

	var req service.AmendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.svc.AmendOrder(c.Request.Context(), user.ID.String(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": order})
}
