package repo

import (
	"context"
	"database/sql"
	
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
)

type WalletRepo struct{ db *sql.DB }

func NewWalletRepo(db *sql.DB) *WalletRepo { return &WalletRepo{db: db} }

// Lock wallet row for update (IMPORTANT)
func (r *WalletRepo) GetForUpdate(ctx context.Context, tx *sql.Tx, userID, assetID string) (*models.Wallet, error) {
	q := `
SELECT id, user_id, asset_id,
       balance::float8,
       in_orders::float8,
       updated_at
FROM wallets
WHERE user_id=$1 AND asset_id=$2
FOR UPDATE`
	row := tx.QueryRowContext(ctx, q, userID, assetID)

	var w models.Wallet
	if err := row.Scan(&w.ID, &w.UserID, &w.AssetID, &w.Balance, &w.InOrders, &w.UpdatedAt); err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepo) UpdateBalances(ctx context.Context, tx *sql.Tx, userID, assetID string, balanceDelta, inOrdersDelta float64) error {
	q := `
		UPDATE wallets
		SET balance = balance + $3,
			in_orders = in_orders + $4,
			updated_at = NOW()
		WHERE user_id=$1 AND asset_id=$2`
	_, err := tx.ExecContext(ctx, q, userID, assetID, balanceDelta, inOrdersDelta)
	return err
}
