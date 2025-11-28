package handler

import (
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/service"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
)

type Handler struct {
	OrderHandler *OrderHandler
	WSHub        *Hub
}

func NewHandler(orderSvc *service.OrderService, marketRepo *repo.MarketRepo) *Handler {
	hub := NewHub(marketRepo)
	go hub.Run()
	go hub.StartCandleBroadcaster()
	
	return &Handler{
		OrderHandler: NewOrderHandler(orderSvc),
		WSHub:        hub,
	}
}
