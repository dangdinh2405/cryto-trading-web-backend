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

// GetCandles retrieves historical candles and aggregates based on interval
func (r *MarketRepo) GetCandles(ctx context.Context, symbol string, interval string, limit int, endTime *time.Time) ([]models.OHLCV, error) {
	// Map interval to minutes
	intervalMap := map[string]int{
		"1m":  1,
		"5m":  5,
		"15m": 15,
		"1h":  60,
		"4h":  240,
		"1D":  1440,
	}

	intervalMinutes, ok := intervalMap[interval]
	if !ok {
		// Default to 1m if invalid interval
		intervalMinutes = 1
	}

	// If 1m requested, query directly
	if intervalMinutes == 1 {
		return r.get1mCandles(ctx, symbol, limit, endTime)
	}

	// For other intervals, fetch more 1m candles and aggregate
	// We need (limit * intervalMinutes) 1m candles to create 'limit' aggregated candles
	requiredCandles := limit * intervalMinutes
	candles1m, err := r.get1mCandles(ctx, symbol, requiredCandles, endTime)
	if err != nil {
		return nil, err
	}

	// Aggregate candles
	return r.aggregateCandles(candles1m, intervalMinutes, limit), nil
}

// get1mCandles fetches 1-minute candles from database
func (r *MarketRepo) get1mCandles(ctx context.Context, symbol string, limit int, endTime *time.Time) ([]models.OHLCV, error) {
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

// aggregateCandles aggregates 1m candles into larger intervals
func (r *MarketRepo) aggregateCandles(candles1m []models.OHLCV, intervalMinutes int, limit int) []models.OHLCV {
	if len(candles1m) == 0 {
		return []models.OHLCV{}
	}

	// Reverse to process in chronological order (oldest first)
	for i, j := 0, len(candles1m)-1; i < j; i, j = i+1, j-1 {
		candles1m[i], candles1m[j] = candles1m[j], candles1m[i]
	}

	var aggregated []models.OHLCV
	intervalDuration := time.Duration(intervalMinutes) * time.Minute

	for i := 0; i < len(candles1m); {
		// Start of this interval (truncate to interval boundary)
		startTime := candles1m[i].OpenTime.Truncate(intervalDuration)
		endTime := startTime.Add(intervalDuration)

		// Aggregate all 1m candles in this interval
		var bucket []models.OHLCV
		for i < len(candles1m) && candles1m[i].OpenTime.Before(endTime) {
			bucket = append(bucket, candles1m[i])
			i++
		}

		if len(bucket) > 0 {
			agg := models.OHLCV{
				Symbol:    bucket[0].Symbol,
				OpenTime:  startTime,
				CloseTime: endTime,
				Open:      bucket[0].Open,
				Close:     bucket[len(bucket)-1].Close,
				High:      bucket[0].High,
				Low:       bucket[0].Low,
				Volume:    0,
			}

			// Calculate high, low, volume
			for _, c := range bucket {
				if c.High > agg.High {
					agg.High = c.High
				}
				if c.Low < agg.Low {
					agg.Low = c.Low
				}
				agg.Volume += c.Volume
			}

			aggregated = append(aggregated, agg)

			// Stop if we have enough candles
			if len(aggregated) >= limit {
				break
			}
		}
	}

	// Reverse back to descending order (newest first)
	for i, j := 0, len(aggregated)-1; i < j; i, j = i+1, j-1 {
		aggregated[i], aggregated[j] = aggregated[j], aggregated[i]
	}

	return aggregated
}

// Trade represents a trade for candle aggregation
type Trade struct {
	Symbol      string
	Price       float64
	Amount      float64
	QuoteAmount float64
	TradeTime   time.Time
}
