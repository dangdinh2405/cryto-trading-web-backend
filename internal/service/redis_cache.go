package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
	"github.com/redis/go-redis/v9"
)

// CacheService handles all Redis caching operations
type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new cache service
func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{client: client}
}

// ============================================================================
// ORDERBOOK CACHING
// ============================================================================

// GetOrderBook retrieves cached orderbook for a market
func (s *CacheService) GetOrderBook(ctx context.Context, marketID string) (*repo.OrderBook, error) {
	key := fmt.Sprintf("orderbook:%s", marketID)
	
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Cache miss
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %v", err)
	}

	var orderbook repo.OrderBook
	if err := json.Unmarshal(data, &orderbook); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %v", err)
	}

	return &orderbook, nil
}

// SetOrderBook caches orderbook data with TTL
func (s *CacheService) SetOrderBook(ctx context.Context, marketID string, orderbook *repo.OrderBook, ttl time.Duration) error {
	key := fmt.Sprintf("orderbook:%s", marketID)
	
	data, err := json.Marshal(orderbook)
	if err != nil {
		return fmt.Errorf("json marshal error: %v", err)
	}

	return s.client.Set(ctx, key, data, ttl).Err()
}

// InvalidateOrderBook removes orderbook from cache
func (s *CacheService) InvalidateOrderBook(ctx context.Context, marketID string) error {
	key := fmt.Sprintf("orderbook:%s", marketID)
	return s.client.Del(ctx, key).Err()
}

// ============================================================================
// MARKET CANDLES CACHING
// ============================================================================

// GetCandle retrieves current candle for a symbol
func (s *CacheService) GetCandle(ctx context.Context, symbol string) (*models.OHLCV, error) {
	key := fmt.Sprintf("candle:%s:current", symbol)
	
	data, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis hgetall error: %v", err)
	}
	
	if len(data) == 0 {
		// No candle exists
		return nil, nil
	}

	// Parse the hash fields into OHLCV struct
	candle, err := s.parseCandle(data, symbol)
	if err != nil {
		return nil, err
	}

	return candle, nil
}

// SetCandle stores current candle for a symbol
func (s *CacheService) SetCandle(ctx context.Context, symbol string, candle *models.OHLCV) error {
	key := fmt.Sprintf("candle:%s:current", symbol)
	
	// Store as Redis hash for efficient updates
	data := map[string]interface{}{
		"symbol":     candle.Symbol,
		"open_time":  candle.OpenTime.Unix(),
		"close_time": candle.CloseTime.Unix(),
		"open":       candle.Open,
		"high":       candle.High,
		"low":        candle.Low,
		"close":      candle.Close,
		"volume":     candle.Volume,
	}

	pipe := s.client.Pipeline()
	pipe.HSet(ctx, key, data)
	pipe.Expire(ctx, key, 90*time.Second) // Auto-expire stale candles
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline error: %v", err)
	}

	// Add to active candles set
	s.client.SAdd(ctx, "candles:active", symbol)

	return nil
}

// GetAllCandles retrieves all active candles
func (s *CacheService) GetAllCandles(ctx context.Context) ([]models.OHLCV, error) {
	// Get all active symbols
	symbols, err := s.client.SMembers(ctx, "candles:active").Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers error: %v", err)
	}

	candles := make([]models.OHLCV, 0, len(symbols))
	
	for _, symbol := range symbols {
		candle, err := s.GetCandle(ctx, symbol)
		if err != nil {
			// Log but continue
			continue
		}
		if candle != nil {
			candles = append(candles, *candle)
		}
	}

	return candles, nil
}

// ResetCandle resets a candle for a new time period
func (s *CacheService) ResetCandle(ctx context.Context, symbol string, newMinute time.Time, lastClose float64) error {
	candle := &models.OHLCV{
		Symbol:    symbol,
		OpenTime:  newMinute,
		CloseTime: newMinute.Add(time.Minute),
		Open:      lastClose,
		High:      lastClose,
		Low:       lastClose,
		Close:     lastClose,
		Volume:    0,
	}
	
	return s.SetCandle(ctx, symbol, candle)
}

// UpdateCandleWithTrade updates an existing candle with new trade data
func (s *CacheService) UpdateCandleWithTrade(ctx context.Context, trade repo.Trade, currentMinute time.Time) error {
	// Get existing candle
	candle, err := s.GetCandle(ctx, trade.Symbol)
	if err != nil {
		return err
	}

	if candle == nil {
		// Create new candle
		candle = &models.OHLCV{
			Symbol:    trade.Symbol,
			OpenTime:  currentMinute,
			CloseTime: currentMinute.Add(time.Minute),
			Open:      trade.Price,
			High:      trade.Price,
			Low:       trade.Price,
			Close:     trade.Price,
			Volume:    trade.QuoteAmount,
		}
	} else {
		// Update existing candle
		if trade.Price > candle.High {
			candle.High = trade.Price
		}
		if trade.Price < candle.Low {
			candle.Low = trade.Price
		}
		candle.Close = trade.Price
		candle.Volume += trade.QuoteAmount
	}

	return s.SetCandle(ctx, trade.Symbol, candle)
}

// HasCandle checks if a candle exists for a symbol
func (s *CacheService) HasCandle(ctx context.Context, symbol string) (bool, error) {
	key := fmt.Sprintf("candle:%s:current", symbol)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// RemoveStaleCandles removes candles for inactive symbols
func (s *CacheService) RemoveStaleCandles(ctx context.Context, activeSymbols []string) error {
	// Get all tracked symbols
	allSymbols, err := s.client.SMembers(ctx, "candles:active").Result()
	if err != nil {
		return err
	}

	activeMap := make(map[string]bool)
	for _, symbol := range activeSymbols {
		activeMap[symbol] = true
	}

	// Remove stale symbols
	for _, symbol := range allSymbols {
		if !activeMap[symbol] {
			key := fmt.Sprintf("candle:%s:current", symbol)
			s.client.Del(ctx, key)
			s.client.SRem(ctx, "candles:active", symbol)
		}
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *CacheService) parseCandle(data map[string]string, symbol string) (*models.OHLCV, error) {
	candle := &models.OHLCV{Symbol: symbol}

	// Parse timestamps
	if openTimeStr, ok := data["open_time"]; ok {
		var openTime int64
		fmt.Sscanf(openTimeStr, "%d", &openTime)
		candle.OpenTime = time.Unix(openTime, 0)
	}
	if closeTimeStr, ok := data["close_time"]; ok {
		var closeTime int64
		fmt.Sscanf(closeTimeStr, "%d", &closeTime)
		candle.CloseTime = time.Unix(closeTime, 0)
	}

	// Parse float values
	fmt.Sscanf(data["open"], "%f", &candle.Open)
	fmt.Sscanf(data["high"], "%f", &candle.High)
	fmt.Sscanf(data["low"], "%f", &candle.Low)
	fmt.Sscanf(data["close"], "%f", &candle.Close)
	fmt.Sscanf(data["volume"], "%f", &candle.Volume)

	return candle, nil
}
