package models

type Market struct {
	ID           string  `json:"id"`
	BaseAssetID  string  `json:"base_asset_id"`
	QuoteAssetID string  `json:"quote_asset_id"`
	Symbol       string  `json:"symbol"`
	MinPrice     float64 `json:"min_price"`
	MaxPrice     float64 `json:"max_price"`
	TickSize     float64 `json:"tick_size"`
	MinNotional  float64 `json:"min_notional"`
	IsActive     bool    `json:"is_active"`
}
