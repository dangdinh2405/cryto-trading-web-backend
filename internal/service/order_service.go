package service

import (
	"context"
	"database/sql"
	"errors"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"

	"math"
)

type OrderService struct {
	db      *sql.DB
	market  *repo.MarketRepo
	order   *repo.OrderRepo
	trade   *repo.TradeRepo
	wallet  *repo.WalletRepo
	feeRate float64
}

func NewOrderService(db *sql.DB, mr *repo.MarketRepo, or *repo.OrderRepo, tr *repo.TradeRepo, wr *repo.WalletRepo) *OrderService {
	return &OrderService{
		db: db, market: mr, order: or, trade: tr, wallet: wr,
		feeRate: 0.001,
	}
}

// ---------------- LIST ORDERS ----------------
func (s *OrderService) ListOrders(ctx context.Context, userID string, status string) ([]*models.Order, error) {
	return s.order.GetByUserID(ctx, userID, status)
}

// ---------------- PLACE ORDER ----------------
func (s *OrderService) PlaceOrder(ctx context.Context, userID string, req PlaceOrderReq) (*models.Order, []*models.Trade, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil { return nil, nil, err }
	defer tx.Rollback()

	market, err := s.market.GetByID(ctx, tx, req.MarketID)
	if err != nil { return nil, nil, err }

	// validate
	if req.Type == models.OrderTypeMarket && req.Side == models.Buy {
		// Market buy only needs quote_amount_max
		if req.QuoteAmountMax == nil || *req.QuoteAmountMax <= 0 {
			return nil, nil, errors.New("market buy requires quote_amount_max > 0")
		}
	} else {
		// All other orders need amount
		if req.Amount <= 0 { return nil, nil, errors.New("amount must be > 0") }
	}
	if req.Type == models.OrderTypeLimit && req.Price == nil {
		return nil, nil, errors.New("limit order requires price")
	}
	if req.Type == models.OrderTypeMarket  && req.Price != nil {
		return nil, nil, errors.New("market order must not have price")
	}
	
	// Market orders cannot be GTC (they must execute immediately or cancel)
	if req.Type == models.OrderTypeMarket && req.TIF == models.GTC {
		return nil, nil, errors.New("market orders cannot use GTC (use IOC or FOK)")
	}

	// POST_ONLY precheck
	if req.TIF == models.PostOnly && req.Type == models.OrderTypeLimit {
		will, err := s.willMatchImmediately(ctx, tx, req, market)
		if err != nil { return nil, nil, err }
		if will { return nil, nil, errors.New("post-only would take liquidity") }
	}

	// lock funds
	if err := s.lockFunds(ctx, tx, market, userID, req); err != nil {
		return nil, nil, err
	}

	// insert taker order
	// For market buy, Amount will be nil initially and updated during matching
	var amount *float64
	if req.Type == models.OrderTypeMarket && req.Side == models.Buy {
		amount = nil // Will be calculated during matching
	} else {
		amount = &req.Amount
	}
	taker := &models.Order{
		UserID: userID, MarketID: req.MarketID,
		Side: req.Side, Type: req.Type, Price: req.Price,
		Amount: amount, FilledAmount: 0,
		QuoteAmountMax: req.QuoteAmountMax,
		Status: models.Open, Fee: 0, TIF: req.TIF,
	}
	if err := s.order.Insert(ctx, tx, taker); err != nil {
		return nil, nil, err
	}

	// match
	trades, err := s.match(ctx, tx, market, taker)
	if err != nil { return nil, nil, err }

	// apply TIF leftover
	if err := s.applyTIF(ctx, tx, market, taker); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil { return nil, nil, err }
	return taker, trades, nil
}

func (s *OrderService) lockFunds(ctx context.Context, tx *sql.Tx, market *models.Market, userID string, req PlaceOrderReq) error {
	base := market.BaseAssetID
	quote := market.QuoteAssetID

	switch req.Side {
	case models.Buy:
		if req.Type == models.OrderTypeMarket{
			// Market buy requires quote_amount_max
			if req.QuoteAmountMax == nil {
				return errors.New("MVP: market buy chưa implement, cần quote_amount_max")
			}
			cost := *req.QuoteAmountMax
			w, err := s.wallet.GetForUpdate(ctx, tx, userID, quote)
			if err != nil { return err }
			if w.Balance < cost { return errors.New("insufficient quote balance") }

			return s.wallet.UpdateBalances(ctx, tx, userID, quote, -cost, +cost)
		}

		cost := (*req.Price) * req.Amount
		w, err := s.wallet.GetForUpdate(ctx, tx, userID, quote)
		if err != nil { return err }
		if w.Balance < cost { return errors.New("insufficient quote balance") }

		return s.wallet.UpdateBalances(ctx, tx, userID, quote, -cost, +cost)

	case models.Sell:
		w, err := s.wallet.GetForUpdate(ctx, tx, userID, base)
		if err != nil { return err }
		if w.Balance < req.Amount { return errors.New("insufficient base balance") }

		return s.wallet.UpdateBalances(ctx, tx, userID, base, -req.Amount, +req.Amount)
	}
	return nil
}

// ---------------- MATCHING ----------------
func (s *OrderService) match(ctx context.Context, tx *sql.Tx, market *models.Market, taker *models.Order) ([]*models.Trade, error) {
	var out []*models.Trade
	var totalQuoteSpent float64 // Track total quote spent for market buy
	var totalBaseBought float64 // Track total base bought for market buy

	// For market buy: continue while we have quote budget remaining
	// For others: continue while we have amount remaining to fill
	for {
		// Check stopping condition
		if taker.Type == models.OrderTypeMarket && taker.Side == models.Buy {
			if taker.QuoteAmountMax == nil || totalQuoteSpent >= *taker.QuoteAmountMax-1e-8 {
				break
			}
		} else {
			if taker.Amount == nil || *taker.Amount-taker.FilledAmount <= 1e-12 {
				break
			}
		}

		makers, err := s.order.SelectMakersForUpdate(ctx, tx, taker.MarketID, taker.Side, taker.Type, taker.Price, 50)
		if err != nil { return nil, err }
		if len(makers) == 0 { break }

		for _, maker := range makers {
			// Check stopping condition again for each maker
			if taker.Type == models.OrderTypeMarket && taker.Side == models.Buy {
				if totalQuoteSpent >= *taker.QuoteAmountMax-1e-8 {
					break
				}
			} else {
				if taker.Amount == nil || *taker.Amount-taker.FilledAmount <= 1e-12 {
					break
				}
			}

			if maker.Amount == nil {
				continue // Skip makers without amount
			}
			makerRem := *maker.Amount - maker.FilledAmount
			tradePrice := *maker.Price // maker price

			var tradeAmt float64
			if taker.Type == models.OrderTypeMarket && taker.Side == models.Buy {
				// For market buy: calculate max amount we can buy with remaining quote
				quoteRemaining := *taker.QuoteAmountMax - totalQuoteSpent
				maxAmtByQuote := quoteRemaining / tradePrice
				tradeAmt = math.Min(maxAmtByQuote, makerRem)
			} else {
				// For normal orders: match by amount
				takerRem := *taker.Amount - taker.FilledAmount
				tradeAmt = math.Min(takerRem, makerRem)
			}

			if tradeAmt <= 1e-12 { break }

			quoteAmt := tradePrice * tradeAmt

			feeTaker := quoteAmt * s.feeRate
			feeMaker := quoteAmt * s.feeRate * 0.5

			tr := &models.Trade{
				MarketID: taker.MarketID,
				MakerOrderID: maker.ID,
				TakerOrderID: taker.ID,
				TakerSide: taker.Side,
				Price: tradePrice,
				Amount: tradeAmt,
				QuoteAmount: quoteAmt,
				FeeMaker: feeMaker,
				FeeTaker: feeTaker,
			}
			if err := s.trade.Insert(ctx, tx, tr); err != nil { return nil, err }

			// update fills
			maker.FilledAmount += tradeAmt
			taker.FilledAmount += tradeAmt
			
			// For market buy, track total bought and quote spent
			if taker.Type == models.OrderTypeMarket && taker.Side == models.Buy {
				totalBaseBought += tradeAmt
				totalQuoteSpent += quoteAmt
			}

			makerStatus := models.PartiallyFilled
			if *maker.Amount >= maker.FilledAmount-1e-12 { makerStatus = models.Filled }
			takerStatus := models.PartiallyFilled
			if taker.Type == models.OrderTypeMarket && taker.Side == models.Buy {
				// For market buy, check if we've spent all our quote budget
				if totalQuoteSpent >= *taker.QuoteAmountMax-1e-8 {
					takerStatus = models.Filled
				}
			} else {
				if taker.Amount != nil && taker.FilledAmount >= *taker.Amount-1e-12 { takerStatus = models.Filled }
			}

			if err := s.order.UpdateFill(ctx, tx, maker.ID, maker.FilledAmount, makerStatus); err != nil { return nil, err }
			if err := s.order.UpdateFill(ctx, tx, taker.ID, taker.FilledAmount, takerStatus); err != nil { return nil, err }

			// settle wallets
			if err := s.settle(ctx, tx, market, maker, taker, tradePrice, tradeAmt, feeMaker, feeTaker); err != nil {
				return nil, err
			}

			out = append(out, tr)
		}
	}
	
	// For market buy, update the Amount field to reflect total bought
	if taker.Type == models.OrderTypeMarket && taker.Side == models.Buy && totalBaseBought > 0 {
		taker.Amount = &totalBaseBought
	}
	
	return out, nil
}

func (s *OrderService) settle(ctx context.Context, tx *sql.Tx, market *models.Market, maker, taker *models.Order,
	price, amount, feeMaker, feeTaker float64) error {

	base := market.BaseAssetID
	quote := market.QuoteAssetID
	cost := price * amount

	if taker.Side == models.Buy {
		var lockedCost, refund float64
		if taker.Type == models.OrderTypeMarket {
			// For market buy, we locked quote_amount_max upfront
			// Unlock the actual cost and refund will be handled in applyTIF
			lockedCost = cost
			refund = 0
		} else {
			// For limit buy, refund difference between limit price and trade price
			lockedCost = (*taker.Price) * amount
			refund = lockedCost - cost
		}

		// taker quote
		if err := s.wallet.UpdateBalances(ctx, tx, taker.UserID, quote, +refund, -lockedCost); err != nil { return err }
		// taker base
		if err := s.wallet.UpdateBalances(ctx, tx, taker.UserID, base, +amount, 0); err != nil { return err }

		// maker sell: base unlock
		if err := s.wallet.UpdateBalances(ctx, tx, maker.UserID, base, 0, -amount); err != nil { return err }
		// maker quote receive
		if err := s.wallet.UpdateBalances(ctx, tx, maker.UserID, quote, +(cost-feeMaker), 0); err != nil { return err }

	} else { // taker SELL
		// taker base unlock
		if err := s.wallet.UpdateBalances(ctx, tx, taker.UserID, base, 0, -amount); err != nil { return err }
		// taker quote receive
		if err := s.wallet.UpdateBalances(ctx, tx, taker.UserID, quote, +(cost-feeTaker), 0); err != nil { return err }

		// maker buy: spend quote locked, refund diff
		lockedCost := (*maker.Price) * amount
		refund := lockedCost - cost

		if err := s.wallet.UpdateBalances(ctx, tx, maker.UserID, quote, +refund, -lockedCost); err != nil { return err }
		if err := s.wallet.UpdateBalances(ctx, tx, maker.UserID, base, +amount, 0); err != nil { return err }
	}
	return nil
}

// ---------------- APPLY TIF ----------------
func (s *OrderService) applyTIF(ctx context.Context, tx *sql.Tx, market *models.Market, taker *models.Order) error {
	if taker.Amount == nil {
		return nil // No remaining for market buy without amount
	}
	remaining := *taker.Amount - taker.FilledAmount
	if remaining <= 1e-12 { return nil }

	// Apply TIF logic (market orders also follow TIF)
	switch taker.TIF {
	case models.GTC:
		// Market orders cannot be GTC in practice, but if somehow it happens, cancel it
		if taker.Type == models.OrderTypeMarket {
			return s.refundRemaining(ctx, tx, market, taker, remaining, models.Canceled)
		}
		return nil
	case models.IOC:
		return s.refundRemaining(ctx, tx, market, taker, remaining, models.Canceled)
	case models.FOK:
		return s.refundRemaining(ctx, tx, market, taker, remaining, models.Rejected)
	case models.PostOnly:
		// POST_ONLY should have been checked earlier, but handle gracefully
		return s.refundRemaining(ctx, tx, market, taker, remaining, models.Canceled)
	}
	return nil
}

func (s *OrderService) refundRemaining(ctx context.Context, tx *sql.Tx, market *models.Market, o *models.Order,
	remaining float64, finalStatus models.OrderStatus) error {

	base := market.BaseAssetID
	quote := market.QuoteAssetID

	if o.Side == models.Buy {
		var refund float64
		if o.Type == models.OrderTypeMarket {
			// For market buy, calculate total quote spent from trades
			if o.QuoteAmountMax != nil {
				// Query total quote spent by this order
				var totalQuoteSpent float64
				q := `SELECT COALESCE(SUM(quote_amount), 0) FROM trades WHERE taker_order_id = $1`
				if err := tx.QueryRowContext(ctx, q, o.ID).Scan(&totalQuoteSpent); err != nil {
					return err
				}
				// Refund the difference between locked and spent
				refund = *o.QuoteAmountMax - totalQuoteSpent
			} else {
				refund = 0
			}
		} else {
			refund = (*o.Price) * remaining
		}
		if err := s.wallet.UpdateBalances(ctx, tx, o.UserID, quote, +refund, -refund); err != nil { return err }
	} else {
		if err := s.wallet.UpdateBalances(ctx, tx, o.UserID, base, +remaining, -remaining); err != nil { return err }
	}
	return s.order.UpdateFill(ctx, tx, o.ID, o.FilledAmount, finalStatus)
}

// MVP: POST_ONLY precheck nhanh
func (s *OrderService) willMatchImmediately(ctx context.Context, tx *sql.Tx, req PlaceOrderReq, market *models.Market) (bool, error) {
	// fetch best opposite price (không lock)
	var q string
	if req.Side == models.Buy {
		q = `
SELECT price FROM orders
WHERE market_id=$1 AND side='sell' AND type='limit'
  AND status IN ('open','partially_filled')
ORDER BY price ASC, created_at ASC
LIMIT 1`
	} else {
		q = `
SELECT price FROM orders
WHERE market_id=$1 AND side='buy' AND type='limit'
  AND status IN ('open','partially_filled')
ORDER BY price DESC, created_at ASC
LIMIT 1`
	}

	var best float64
	err := tx.QueryRowContext(ctx, q, req.MarketID).Scan(&best)
	if err == sql.ErrNoRows { return false, nil }
	if err != nil { return false, err }

	if req.Side == models.Buy {
		return *req.Price >= best, nil
	}
	return *req.Price <= best, nil
}

// ---------------- CANCEL ORDER ----------------
func (s *OrderService) CancelOrder(ctx context.Context, userID, orderID string) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil { return err }
	defer tx.Rollback()

	o, err := s.order.GetByIDForUpdate(ctx, tx, orderID)
	if err != nil { return err }
	if o.UserID != userID { return errors.New("forbidden") }
	if o.Status != models.Open && o.Status != models.PartiallyFilled {
		return errors.New("cannot cancel in this status")
	}

	market, err := s.market.GetByID(ctx, tx, o.MarketID)
	if err != nil { return err }

	if o.Amount != nil {
		remaining := *o.Amount - o.FilledAmount
		if remaining > 1e-12 {
			if err := s.refundRemaining(ctx, tx, market, o, remaining, models.Canceled); err != nil {
				return err
			}
		}
	}
	if err := s.order.Cancel(ctx, tx, o.ID); err != nil { 
		return err 
	}
	
	return tx.Commit()
}

// ---------------- AMEND ORDER ----------------
func (s *OrderService) AmendOrder(ctx context.Context, userID, orderID string, req AmendReq) (*models.Order, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil { return nil, err }
	defer tx.Rollback()

	o, err := s.order.GetByIDForUpdate(ctx, tx, orderID)
	if err != nil { return nil, err }
	if o.UserID != userID { return nil, errors.New("forbidden") }
	if o.Type != models.OrderTypeLimit { return nil, errors.New("only limit amendable") }
	if o.Status != models.Open && o.Status != models.PartiallyFilled {
		return nil, errors.New("cannot amend in this status")
	}
	if o.Amount == nil {
		return nil, errors.New("cannot amend market buy order")
	}
	if req.NewAmount < o.FilledAmount {
		return nil, errors.New("new amount < filled")
	}

	market, err := s.market.GetByID(ctx, tx, o.MarketID)
	if err != nil { return nil, err }

	newPrice := o.Price
	if req.NewPrice != nil { newPrice = req.NewPrice }

	oldRem := *o.Amount - o.FilledAmount
	newRem := req.NewAmount - o.FilledAmount
	deltaRem := newRem - oldRem

	base := market.BaseAssetID
	quote := market.QuoteAssetID

	if math.Abs(deltaRem) > 1e-12 {
		if o.Side == models.Buy {
			deltaQuote := (*newPrice) * deltaRem
			w, err := s.wallet.GetForUpdate(ctx, tx, userID, quote)
			if err != nil { return nil, err }
			if deltaQuote > 0 && w.Balance < deltaQuote {
				return nil, errors.New("insufficient quote for amend")
			}
			if err := s.wallet.UpdateBalances(ctx, tx, userID, quote, -deltaQuote, +deltaQuote); err != nil {
				return nil, err
			}
		} else {
			deltaBase := deltaRem
			w, err := s.wallet.GetForUpdate(ctx, tx, userID, base)
			if err != nil { return nil, err }
			if deltaBase > 0 && w.Balance < deltaBase {
				return nil, errors.New("insufficient base for amend")
			}
			if err := s.wallet.UpdateBalances(ctx, tx, userID, base, -deltaBase, +deltaBase); err != nil {
				return nil, err
			}
		}
	}

	// update order row
	q := `UPDATE orders SET price=$2, amount=$3, updated_at=NOW() WHERE id=$1`
	if _, err := tx.ExecContext(ctx, q, o.ID, newPrice, req.NewAmount); err != nil {
		return nil, err
	}

	o.Price = newPrice
	o.Amount = &req.NewAmount

	if err := tx.Commit(); err != nil { return nil, err }
	return o, nil
}
