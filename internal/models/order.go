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
	ID             string      `json:"id"`
	UserID         string      `json:"userId"`
	MarketID       string      `json:"marketId"`
	Symbol         string      `json:"symbol"` // From markets table JOIN
	Side           OrderSide   `json:"side"`
	Type           OrderType   `json:"type"`
	Price          *float64    `json:"price"`          // nil nếu market
	Amount         *float64    `json:"amount"`         // nil for market buy initially
	FilledAmount   float64     `json:"filled"`
	QuoteAmountMax *float64    `json:"quoteAmountMax"` // for market buy only
	Status         OrderStatus `json:"status"`
	Fee            float64     `json:"fee"`
	TIF            TimeInForce `json:"tif"`

	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	CanceledAt *time.Time `json:"canceledAt,omitempty"`
}

