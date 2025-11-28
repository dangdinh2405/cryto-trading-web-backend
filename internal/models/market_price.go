package models

import "time"

// MarketPriceLive for real-time streaming (only current price)
type MarketPriceLive struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}
