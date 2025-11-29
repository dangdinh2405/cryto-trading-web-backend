package handler

import (
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/service"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
)

type Handler struct {
	OrderHandler  *OrderHandler
	MarketHandler *MarketHandler
	WSHub         *Hub
	OrderbookHub  *OrderbookHub
}

func NewHandler(orderSvc *service.OrderService, marketRepo *repo.MarketRepo, orderRepo *repo.OrderRepo) *Handler {
	hub := NewHub(marketRepo)
	go hub.Run()
	go hub.StartCandleBroadcaster()

	orderbookHub := NewOrderbookHub(orderRepo)
	go orderbookHub.Run()
	go orderbookHub.StartOrderbookBroadcaster(marketRepo)
	
	return &Handler{
		OrderHandler:  NewOrderHandler(orderSvc),
		MarketHandler: NewMarketHandler(marketRepo),
		WSHub:         hub,
		OrderbookHub:  orderbookHub,
	}
}
