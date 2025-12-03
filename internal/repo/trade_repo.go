package repo

import (
	"context"
	"database/sql"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
)

type TradeRepo struct{ db *sql.DB }
func NewTradeRepo(db *sql.DB) *TradeRepo { return &TradeRepo{db: db} }

func (r *TradeRepo) Insert(ctx context.Context, tx *sql.Tx, t *models.Trade) error {
	q := `
		INSERT INTO trades(market_id, maker_order_id, taker_order_id, taker_side, price, amount, quote_amount, fee_maker, fee_taker)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING id, trade_time`
	return tx.QueryRowContext(ctx, q,
		t.MarketID, t.MakerOrderID, t.TakerOrderID, t.TakerSide,
		t.Price, t.Amount, t.QuoteAmount, t.FeeMaker, t.FeeTaker,
	).Scan(&t.ID, &t.TradeTime)
}

// TradeWithSymbol includes market symbol for API response
type TradeWithSymbol struct {
	ID          string
	Symbol      string
	Side        models.OrderSide
	Price       float64
	Amount      float64
	QuoteAmount float64
	Fee         float64
	TradeTime   sql.NullTime
}

// GetByUserId returns all trades for a user (from both maker and taker orders)
func (r *TradeRepo) GetByUserId(ctx context.Context, userId string, limit int) ([]TradeWithSymbol, error) {
	if limit <= 0 {
		limit = 100
	}

	q := `
		SELECT 
			t.id,
			m.symbol,
			CASE 
				WHEN o.user_id = $1 AND o.side = 'buy' THEN 'buy'
				WHEN o.user_id = $1 AND o.side = 'sell' THEN 'sell'
				WHEN o2.user_id = $1 AND o2.side = 'buy' THEN 'buy'
				ELSE 'sell'
			END as side,
			t.price,
			t.amount,
			t.quote_amount,
			CASE 
				WHEN o.user_id = $1 THEN 
					CASE WHEN t.taker_order_id = o.id THEN t.fee_taker ELSE t.fee_maker END
				ELSE 
					CASE WHEN t.taker_order_id = o2.id THEN t.fee_taker ELSE t.fee_maker END
			END as fee,
			t.trade_time
		FROM trades t
		JOIN markets m ON t.market_id = m.id
		LEFT JOIN orders o ON (t.maker_order_id = o.id OR t.taker_order_id = o.id) AND o.user_id = $1
		LEFT JOIN orders o2 ON (t.maker_order_id = o2.id OR t.taker_order_id = o2.id) AND o2.user_id = $1
		WHERE (o.user_id = $1 OR o2.user_id = $1)
		ORDER BY t.trade_time DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, q, userId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []TradeWithSymbol
	for rows.Next() {
		var t TradeWithSymbol
		if err := rows.Scan(&t.ID, &t.Symbol, &t.Side, &t.Price, &t.Amount, &t.QuoteAmount, &t.Fee, &t.TradeTime); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}
