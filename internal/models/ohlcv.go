package models

import "time"

// OHLCV represents a 1-minute candle
type OHLCV struct {
	Symbol    string    `json:"symbol"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"` // Current price (live updates)
	Volume    float64   `json:"volume"`
}
