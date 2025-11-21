package handler

import "github.com/dangdinh2405/cryto-trading-web-backend/internal/service"

type Handler struct {
	OrderHandler *OrderHandler
}

func NewHandler(orderSvc *service.OrderService) *Handler {
	return &Handler{
		OrderHandler: NewOrderHandler(orderSvc),
	}
}
