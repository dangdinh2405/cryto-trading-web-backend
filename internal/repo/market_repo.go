package repo

import (
	"context"
	"database/sql"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
)

type MarketRepo struct{ db *sql.DB }
func NewMarketRepo(db *sql.DB) *MarketRepo { return &MarketRepo{db: db} }

func (r *MarketRepo) GetByID(ctx context.Context, tx *sql.Tx, id string) (*models.Market, error) {
	q := `SELECT id, symbol, base_asset_id, quote_asset_id FROM markets WHERE id=$1`
	row := tx.QueryRowContext(ctx, q, id)

	var m models.Market
	if err := row.Scan(&m.ID, &m.Symbol, &m.BaseAssetID, &m.QuoteAssetID); err != nil {
		return nil, err
	}
	return &m, nil
}
