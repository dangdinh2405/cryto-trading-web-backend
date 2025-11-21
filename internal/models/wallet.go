package models

import "time"

type Wallet struct {
	ID        string
	UserID    string
	AssetID   string
	Balance   float64
	InOrders  float64
	UpdatedAt time.Time
}
