package handler

import "github.com/gin-gonic/gin"

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	orders := r.Group("/orders")
	{
		orders.POST("", h.OrderHandler.Place)
		orders.DELETE("/:id", h.OrderHandler.Cancel)
		orders.PUT("/:id", h.OrderHandler.Amend)
	}
}