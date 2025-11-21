package service

import "github.com/dangdinh2405/cryto-trading-web-backend/internal/models"

type PlaceOrderReq struct {
	MarketID string           `json:"marketId" binding:"required"`
	Side     models.OrderSide `json:"side" binding:"required,oneof=buy sell"`
	Type     models.OrderType `json:"type" binding:"required,oneof=market limit"`
	Price    *float64         `json:"price,omitempty"`
	Amount   float64          `json:"amount" binding:"required,gt=0"`
	TIF      models.TimeInForce `json:"tif" binding:"required,oneof=GTC IOC FOK POST_ONLY"`
}

type AmendReq struct {
	NewPrice  *float64 `json:"price,omitempty"`
	NewAmount float64  `json:"amount" binding:"required,gt=0"`
}
