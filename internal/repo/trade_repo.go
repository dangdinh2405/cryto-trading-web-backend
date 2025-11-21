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
