package models

import "time"

type Trade struct {
	ID           string
	MarketID     string
	MakerOrderID string
	TakerOrderID string
	TakerSide    OrderSide
	Price        float64
	Amount       float64
	QuoteAmount  float64
	FeeMaker     float64
	FeeTaker     float64
	TradeTime    time.Time
}
