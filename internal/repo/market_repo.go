package repo

import (
	"context"
	"time"
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

// GetAllActiveMarkets retrieves all active markets
func (r *MarketRepo) GetAllActiveMarkets(ctx context.Context) ([]models.Market, error) {
	q := `SELECT id, symbol, base_asset_id, quote_asset_id FROM markets WHERE is_active = true`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []models.Market
	for rows.Next() {
		var m models.Market
		if err := rows.Scan(&m.ID, &m.Symbol, &m.BaseAssetID, &m.QuoteAssetID); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

// GetLatestTrades gets recent trades for candle aggregation
func (r *MarketRepo) GetLatestTrades(ctx context.Context, since time.Time) ([]Trade, error) {
	q := `
		SELECT 
			m.symbol,
			t.price,
			t.amount,
			t.quote_amount,
			t.trade_time
		FROM trades t
		JOIN markets m ON t.market_id = m.id
		WHERE t.trade_time >= $1
		AND m.is_active = true
		ORDER BY t.trade_time ASC
	`

	rows, err := r.db.QueryContext(ctx, q, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var trade Trade
		if err := rows.Scan(
			&trade.Symbol,
			&trade.Price,
			&trade.Amount,
			&trade.QuoteAmount,
			&trade.TradeTime,
		); err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}
	return trades, rows.Err()
}

// SaveOHLCV saves completed candle to database
func (r *MarketRepo) SaveOHLCV(ctx context.Context, candle *models.OHLCV) error {
	q := `
		INSERT INTO ohlcv_1m (symbol, open_time, close_time, open, high, low, close, volume)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (symbol, open_time) DO UPDATE SET
			close_time = EXCLUDED.close_time,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume
	`

	_, err := r.db.ExecContext(ctx, q,
		candle.Symbol,
		candle.OpenTime,
		candle.CloseTime,
		candle.Open,
		candle.High,
		candle.Low,
		candle.Close,
		candle.Volume,
	)
	return err
}

// Trade represents a trade for candle aggregation
type Trade struct {
	Symbol      string
	Price       float64
	Amount      float64
	QuoteAmount float64
	TradeTime   time.Time
}
