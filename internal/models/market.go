package models

type Market struct {
	ID           string
	BaseAssetID  string
	QuoteAssetID string
	Symbol	     string
	MinPrice     float64
	MaxPrice     float64
	TickSize	 float64
	MinNotional  float64
	IsActive     bool
}
