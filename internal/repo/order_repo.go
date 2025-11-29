package repo

import (
	"context"
	"database/sql"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
)

type OrderRepo struct{ db *sql.DB }
func NewOrderRepo(db *sql.DB) *OrderRepo { return &OrderRepo{db: db} }

func (r *OrderRepo) Insert(ctx context.Context, tx *sql.Tx, o *models.Order) error {
	q := `
INSERT INTO orders(user_id, market_id, side, type, price, amount, filled_amount, quote_amount_max, status, fee, tif)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id, created_at, updated_at`
	return tx.QueryRowContext(ctx, q,
		o.UserID, o.MarketID, o.Side, o.Type, o.Price,
		o.Amount, o.FilledAmount, o.QuoteAmountMax, o.Status, o.Fee, o.TIF,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (r *OrderRepo) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*models.Order, error) {
	q := `
SELECT id,user_id,market_id,side,type,price,amount,filled_amount,quote_amount_max,status,fee,tif,created_at,updated_at,canceled_at
FROM orders WHERE id=$1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, q, id)

	var o models.Order
	if err := row.Scan(&o.ID, &o.UserID, &o.MarketID, &o.Side, &o.Type, &o.Price,
			&o.Amount, &o.FilledAmount, &o.QuoteAmountMax, &o.Status, &o.Fee, &o.TIF,
			&o.CreatedAt, &o.UpdatedAt, &o.CanceledAt); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *OrderRepo) UpdateFill(ctx context.Context, tx *sql.Tx, id string, newFilled float64, newStatus models.OrderStatus) error {
	q := `UPDATE orders SET filled_amount=$2, status=$3, updated_at=NOW() WHERE id=$1`
	_, err := tx.ExecContext(ctx, q, id, newFilled, newStatus)
	return err
}

func (r *OrderRepo) Cancel(ctx context.Context, tx *sql.Tx, id string) error {
	q := `UPDATE orders SET status='canceled', canceled_at=NOW(), updated_at=NOW() WHERE id=$1`
	_, err := tx.ExecContext(ctx, q, id)
	return err
}

// OrderBookEntry represents a price level in the orderbook
type OrderBookEntry struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"` // Total amount at this price level
}

// OrderBook represents the full orderbook for a market
type OrderBook struct {
	MarketID string           `json:"market_id"`
	Bids     []OrderBookEntry `json:"bids"` // Buy orders, sorted DESC by price
	Asks     []OrderBookEntry `json:"asks"` // Sell orders, sorted ASC by price
}

// GetOrderBook retrieves aggregated orderbook for a market
// Only includes limit orders with GTC or POST_ONLY TIF
func (r *OrderRepo) GetOrderBook(ctx context.Context, marketID string, limit int) (*OrderBook, error) {
	// Get sell orders (asks) - sorted by price ASC (lowest first)
	asksQuery := `
		SELECT price, SUM(amount - filled_amount) as total_amount
		FROM orders
		WHERE market_id = $1 
			AND side = 'sell' 
			AND type = 'limit'
			AND status IN ('open', 'partially_filled')
			AND tif IN ('GTC', 'POST_ONLY')
			AND price IS NOT NULL
		GROUP BY price
		ORDER BY price ASC
		LIMIT $2
	`

	// Get buy orders (bids) - sorted by price DESC (highest first)
	bidsQuery := `
		SELECT price, SUM(amount - filled_amount) as total_amount
		FROM orders
		WHERE market_id = $1 
			AND side = 'buy' 
			AND type = 'limit'
			AND status IN ('open', 'partially_filled')
			AND tif IN ('GTC', 'POST_ONLY')
			AND price IS NOT NULL
		GROUP BY price
		ORDER BY price DESC
		LIMIT $2
	`

	var asks, bids []OrderBookEntry
	// Initialize as empty slices to return [] instead of null in JSON
	asks = []OrderBookEntry{}
	bids = []OrderBookEntry{}

	// Fetch asks
	asksRows, err := r.db.QueryContext(ctx, asksQuery, marketID, limit)
	if err != nil {
		return nil, err
	}
	defer asksRows.Close()

	for asksRows.Next() {
		var entry OrderBookEntry
		if err := asksRows.Scan(&entry.Price, &entry.Amount); err != nil {
			return nil, err
		}
		asks = append(asks, entry)
	}
	if err := asksRows.Err(); err != nil {
		return nil, err
	}

	// Fetch bids
	bidsRows, err := r.db.QueryContext(ctx, bidsQuery, marketID, limit)
	if err != nil {
		return nil, err
	}
	defer bidsRows.Close()

	for bidsRows.Next() {
		var entry OrderBookEntry
		if err := bidsRows.Scan(&entry.Price, &entry.Amount); err != nil {
			return nil, err
		}
		bids = append(bids, entry)
	}
	if err := bidsRows.Err(); err != nil {
		return nil, err
	}

	return &OrderBook{
		MarketID: marketID,
		Bids:     bids,
		Asks:     asks,
	}, nil
}

// Select makers for matching (locked + skip locked)
func (r *OrderRepo) SelectMakersForUpdate(ctx context.Context, tx *sql.Tx, marketID string, takerSide models.OrderSide, takerType models.OrderType, takerPrice *float64, limit int) ([]*models.Order, error) {
	var q string
	if takerSide == models.Buy {
		q = `
		SELECT id,user_id,market_id,side,type,price,amount,filled_amount,quote_amount_max,status,fee,tif,created_at,updated_at,canceled_at
		FROM orders
		WHERE market_id=$1 AND side='sell' AND type='limit'
		AND status IN ('open','partially_filled')
		AND ($2='market' OR price <= $3)
		ORDER BY price ASC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $4`
	} else {
		q = `
		SELECT id,user_id,market_id,side,type,price,amount,filled_amount,quote_amount_max,status,fee,tif,created_at,updated_at,canceled_at
		FROM orders
		WHERE market_id=$1 AND side='buy' AND type='limit'
		AND status IN ('open','partially_filled')
		AND ($2='market' OR price >= $3)
		ORDER BY price DESC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $4`
	}

	priceVal := 0.0
	if takerPrice != nil { priceVal = *takerPrice }

	rows, err := tx.QueryContext(ctx, q, marketID, takerType, priceVal, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var makers []*models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID,&o.UserID,&o.MarketID,&o.Side,&o.Type,&o.Price,
				&o.Amount,&o.FilledAmount,&o.QuoteAmountMax,&o.Status,&o.Fee,&o.TIF,
				&o.CreatedAt,&o.UpdatedAt,&o.CanceledAt); err != nil {
			return nil, err
		}
		makers = append(makers, &o)
	}
	return makers, rows.Err()
}
