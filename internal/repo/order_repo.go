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
INSERT INTO orders(user_id, market_id, side, type, price, amount, filled_amount, status, fee, tif)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
RETURNING id, created_at, updated_at`
	return tx.QueryRowContext(ctx, q,
		o.UserID, o.MarketID, o.Side, o.Type, o.Price,
		o.Amount, o.FilledAmount, o.Status, o.Fee, o.TIF,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt)
}

func (r *OrderRepo) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*models.Order, error) {
	q := `
SELECT id,user_id,market_id,side,type,price,amount,filled_amount,status,fee,tif,created_at,updated_at,canceled_at
FROM orders WHERE id=$1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, q, id)

	var o models.Order
	if err := row.Scan(&o.ID, &o.UserID, &o.MarketID, &o.Side, &o.Type, &o.Price,
		&o.Amount, &o.FilledAmount, &o.Status, &o.Fee, &o.TIF,
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

// Select makers for matching (locked + skip locked)
func (r *OrderRepo) SelectMakersForUpdate(ctx context.Context, tx *sql.Tx, marketID string, takerSide models.OrderSide, takerType models.OrderType, takerPrice *float64, limit int) ([]*models.Order, error) {
	var q string
	if takerSide == models.Buy {
		q = `
		SELECT id,user_id,market_id,side,type,price,amount,filled_amount,status,fee,tif,created_at,updated_at,canceled_at
		FROM orders
		WHERE market_id=$1 AND side='sell' AND type='limit'
		AND status IN ('open','partially_filled')
		AND ($2='market' OR price <= $3)
		ORDER BY price ASC, created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $4`
	} else {
		q = `
		SELECT id,user_id,market_id,side,type,price,amount,filled_amount,status,fee,tif,created_at,updated_at,canceled_at
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
			&o.Amount,&o.FilledAmount,&o.Status,&o.Fee,&o.TIF,
			&o.CreatedAt,&o.UpdatedAt,&o.CanceledAt); err != nil {
			return nil, err
		}
		makers = append(makers, &o)
	}
	return makers, rows.Err()
}
