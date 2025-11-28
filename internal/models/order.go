package models

import "time"

type OrderSide string
const (
	Buy  OrderSide = "buy"
	Sell OrderSide = "sell"
)

type OrderType string
const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

type OrderStatus string
const (
	Open            OrderStatus = "open"
	PartiallyFilled OrderStatus = "partially_filled"
	Filled          OrderStatus = "filled"
	Canceled        OrderStatus = "canceled"
	Rejected        OrderStatus = "rejected"
	Expired         OrderStatus = "expired"
)

type TimeInForce string
const (
	GTC      TimeInForce = "GTC"
	IOC      TimeInForce = "IOC"
	FOK      TimeInForce = "FOK"
	PostOnly TimeInForce = "POST_ONLY"
)

type Order struct {
	ID             string
	UserID         string
	MarketID       string
	Side           OrderSide
	Type           OrderType
	Price          *float64 // nil nếu market
	Amount         *float64 // nil for market buy initially
	FilledAmount   float64
	QuoteAmountMax *float64 // for market buy only
	Status         OrderStatus
	Fee            float64
	TIF            TimeInForce

	CreatedAt  time.Time
	UpdatedAt  time.Time
	CanceledAt *time.Time
}
