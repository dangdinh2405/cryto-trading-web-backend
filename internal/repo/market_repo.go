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
	q := `SELECT id, symbol, base_asset_id, quote_asset_id, min_price, max_price, tick_size, min_notional FROM markets WHERE is_active = true`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []models.Market
	for rows.Next() {
		var m models.Market
		if err := rows.Scan(&m.ID, &m.Symbol, &m.BaseAssetID, &m.QuoteAssetID, &m.MinPrice, &m.MaxPrice, &m.TickSize, &m.MinNotional); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	return markets, rows.Err()
}

// ValidateSymbol checks if a symbol exists in the markets table
func (r *MarketRepo) ValidateSymbol(ctx context.Context, symbol string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM markets WHERE symbol = $1 AND is_active = true)`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, symbol).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
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

// GetCandles retrieves historical candles from ohlcv_1m table
func (r *MarketRepo) GetCandles(ctx context.Context, symbol string, interval string, limit int, endTime *time.Time) ([]models.OHLCV, error) {
	// For now, we only support 1m interval since we only have ohlcv_1m table
	// You can extend this to support other intervals by creating additional tables
	
	var q string
	var args []interface{}
	
	if endTime != nil {
		q = `
			SELECT symbol, open_time, close_time, open, high, low, close, volume
			FROM ohlcv_1m
			WHERE symbol = $1 AND open_time < $2
			ORDER BY open_time DESC
			LIMIT $3
		`
		args = []interface{}{symbol, *endTime, limit}
	} else {
		q = `
			SELECT symbol, open_time, close_time, open, high, low, close, volume
			FROM ohlcv_1m
			WHERE symbol = $1
			ORDER BY open_time DESC
			LIMIT $2
		`
		args = []interface{}{symbol, limit}
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candles []models.OHLCV
	for rows.Next() {
		var c models.OHLCV
		if err := rows.Scan(&c.Symbol, &c.OpenTime, &c.CloseTime, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			return nil, err
		}
		candles = append(candles, c)
	}
	return candles, rows.Err()
}

// Trade represents a trade for candle aggregation
type Trade struct {
	Symbol      string
	Price       float64
	Amount      float64
	QuoteAmount float64
	TradeTime   time.Time
}
