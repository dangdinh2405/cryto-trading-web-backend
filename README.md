# Crypto Trading Platform - API Documentation

## Base URL
```
https://{projectId}.supabase.co/functions/v1/make-server-6b6c8494
```

## Authentication
Most endpoints require authentication using Bearer token in the Authorization header:
```
Authorization: Bearer {access_token}
```

---

## 1. Authentication APIs

### 1.1 Register User
**Endpoint:** `POST /auth/register`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "name": "John Doe"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "user": {
      "id": "uuid-string",
      "email": "user@example.com",
      "name": "John Doe",
      "created_at": "2025-11-17T10:30:00.000Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Error Response (400):**
```json
{
  "success": false,
  "error": "Email already exists"
}
```

---

### 1.2 Login
**Endpoint:** `POST /auth/login`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "user": {
      "id": "uuid-string",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "user",
      "created_at": "2025-11-17T10:30:00.000Z"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

**Error Response (401):**
```json
{
  "success": false,
  "error": "Invalid credentials"
}
```

---

### 1.3 Logout
**Endpoint:** `POST /auth/logout`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

---

### 1.4 Get Current Session
**Endpoint:** `GET /auth/session`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid-string",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "user"
    },
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }
}
```

---

## 2. Market Data APIs

### 2.1 Get Market Overview
**Endpoint:** `GET /market/overview`

**Headers:**
```json
{
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "symbol": "BTCUSDT",
      "name": "Bitcoin",
      "price": 43250.50,
      "change_24h": 2.5,
      "volume_24h": 28500000000,
      "high_24h": 43500.00,
      "low_24h": 42100.00,
      "market_cap": 850000000000
    },
    {
      "symbol": "ETHUSDT",
      "name": "Ethereum",
      "price": 2280.75,
      "change_24h": -1.2,
      "volume_24h": 12000000000,
      "high_24h": 2320.00,
      "low_24h": 2250.00,
      "market_cap": 275000000000
    }
  ]
}
```

---

### 2.2 Get Candlestick Data
**Endpoint:** `GET /market/candles/:symbol`

**Query Parameters:**
- `interval` - Time interval (1m, 5m, 15m, 1h, 4h, 1d)
- `limit` - Number of candles (default: 100, max: 1000)

**Example:** `GET /market/candles/BTCUSDT?interval=1h&limit=100`

**Headers:**
```json
{
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1h",
    "candles": [
      {
        "timestamp": 1700222400000,
        "open": 43100.00,
        "high": 43250.00,
        "low": 43050.00,
        "close": 43200.00,
        "volume": 1250.5
      },
      {
        "timestamp": 1700226000000,
        "open": 43200.00,
        "high": 43350.00,
        "low": 43150.00,
        "close": 43250.50,
        "volume": 980.3
      }
    ]
  }
}
```

---

### 2.3 Get Order Book
**Endpoint:** `GET /market/orderbook/:symbol`

**Query Parameters:**
- `limit` - Number of levels (default: 20)

**Example:** `GET /market/orderbook/BTCUSDT?limit=20`

**Headers:**
```json
{
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "timestamp": 1700226000000,
    "bids": [
      {
        "price": 43250.00,
        "quantity": 2.5,
        "total": 108125.00
      },
      {
        "price": 43249.50,
        "quantity": 1.8,
        "total": 77849.10
      }
    ],
    "asks": [
      {
        "price": 43251.00,
        "quantity": 3.2,
        "total": 138403.20
      },
      {
        "price": 43252.50,
        "quantity": 2.1,
        "total": 90830.25
      }
    ]
  }
}
```

---

### 2.4 Get Recent Trades
**Endpoint:** `GET /market/trades/:symbol`

**Query Parameters:**
- `limit` - Number of trades (default: 50, max: 500)

**Example:** `GET /market/trades/BTCUSDT?limit=50`

**Headers:**
```json
{
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "trades": [
      {
        "id": "trade-uuid-1",
        "price": 43250.50,
        "quantity": 0.5,
        "total": 21625.25,
        "side": "buy",
        "timestamp": 1700226000000
      },
      {
        "id": "trade-uuid-2",
        "price": 43249.00,
        "quantity": 0.3,
        "total": 12974.70,
        "side": "sell",
        "timestamp": 1700225995000
      }
    ]
  }
}
```

---

### 2.5 Get Ticker (Real-time Price)
**Endpoint:** `GET /market/ticker/:symbol`

**Example:** `GET /market/ticker/BTCUSDT`

**Headers:**
```json
{
  "Authorization": "Bearer {publicAnonKey}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "price": 43250.50,
    "bid": 43250.00,
    "ask": 43251.00,
    "change_24h": 2.5,
    "volume_24h": 28500000000,
    "high_24h": 43500.00,
    "low_24h": 42100.00,
    "timestamp": 1700226000000
  }
}
```

---

## 3. Wallet APIs

### 3.1 Get User Wallet Balance
**Endpoint:** `GET /wallet/balance`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "user_id": "uuid-string",
    "balances": [
      {
        "asset": "USDT",
        "available": 8500.50,
        "locked": 1499.50,
        "total": 10000.00
      },
      {
        "asset": "BTC",
        "available": 0.15,
        "locked": 0.05,
        "total": 0.20
      },
      {
        "asset": "ETH",
        "available": 2.5,
        "locked": 0.5,
        "total": 3.0
      }
    ],
    "total_value_usd": 15250.75,
    "updated_at": "2025-11-17T10:30:00.000Z"
  }
}
```

---

### 3.2 Get Wallet History
**Endpoint:** `GET /wallet/history`

**Query Parameters:**
- `asset` - Filter by asset (optional)
- `type` - Filter by type: deposit, withdrawal, trade (optional)
- `limit` - Number of records (default: 50)
- `offset` - Pagination offset (default: 0)

**Example:** `GET /wallet/history?asset=USDT&limit=20`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "history": [
      {
        "id": "history-uuid-1",
        "type": "trade",
        "asset": "USDT",
        "amount": -1000.00,
        "balance_after": 9000.00,
        "description": "Buy 0.025 BTC",
        "timestamp": "2025-11-17T10:25:00.000Z"
      },
      {
        "id": "history-uuid-2",
        "type": "deposit",
        "asset": "USDT",
        "amount": 10000.00,
        "balance_after": 10000.00,
        "description": "Initial demo balance",
        "timestamp": "2025-11-17T10:00:00.000Z"
      }
    ],
    "total": 2,
    "limit": 20,
    "offset": 0
  }
}
```

---

### 3.3 Reset Demo Balance
**Endpoint:** `POST /wallet/reset`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Demo balance reset successfully",
  "data": {
    "balances": [
      {
        "asset": "USDT",
        "available": 10000.00,
        "locked": 0.00,
        "total": 10000.00
      }
    ]
  }
}
```

---

## 4. Order Management APIs

### 4.1 Create Order
**Endpoint:** `POST /orders/create`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {access_token}"
}
```

**Request Body (Market Order):**
```json
{
  "symbol": "BTCUSDT",
  "side": "buy",
  "type": "market",
  "quantity": 0.025
}
```

**Request Body (Limit Order):**
```json
{
  "symbol": "BTCUSDT",
  "side": "sell",
  "type": "limit",
  "quantity": 0.025,
  "price": 43500.00
}
```

**Success Response (201):**
```json
{
  "success": true,
  "message": "Order created successfully",
  "data": {
    "order": {
      "id": "order-uuid-1",
      "user_id": "user-uuid",
      "symbol": "BTCUSDT",
      "side": "buy",
      "type": "market",
      "quantity": 0.025,
      "filled_quantity": 0.025,
      "price": null,
      "average_price": 43250.50,
      "total": 1081.26,
      "status": "filled",
      "created_at": "2025-11-17T10:30:00.000Z",
      "updated_at": "2025-11-17T10:30:00.000Z"
    }
  }
}
```

**Error Response (400):**
```json
{
  "success": false,
  "error": "Insufficient balance"
}
```

---

### 4.2 Get User Orders
**Endpoint:** `GET /orders`

**Query Parameters:**
- `symbol` - Filter by symbol (optional)
- `status` - Filter by status: pending, filled, cancelled, partially_filled (optional)
- `limit` - Number of orders (default: 50)
- `offset` - Pagination offset (default: 0)

**Example:** `GET /orders?symbol=BTCUSDT&status=filled&limit=20`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "orders": [
      {
        "id": "order-uuid-1",
        "symbol": "BTCUSDT",
        "side": "buy",
        "type": "market",
        "quantity": 0.025,
        "filled_quantity": 0.025,
        "price": null,
        "average_price": 43250.50,
        "total": 1081.26,
        "status": "filled",
        "created_at": "2025-11-17T10:30:00.000Z",
        "updated_at": "2025-11-17T10:30:00.000Z"
      },
      {
        "id": "order-uuid-2",
        "symbol": "ETHUSDT",
        "side": "buy",
        "type": "limit",
        "quantity": 1.0,
        "filled_quantity": 0.0,
        "price": 2250.00,
        "average_price": null,
        "total": 2250.00,
        "status": "pending",
        "created_at": "2025-11-17T10:25:00.000Z",
        "updated_at": "2025-11-17T10:25:00.000Z"
      }
    ],
    "total": 2,
    "limit": 20,
    "offset": 0
  }
}
```

---

### 4.3 Get Order Details
**Endpoint:** `GET /orders/:orderId`

**Example:** `GET /orders/order-uuid-1`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "order": {
      "id": "order-uuid-1",
      "user_id": "user-uuid",
      "symbol": "BTCUSDT",
      "side": "buy",
      "type": "market",
      "quantity": 0.025,
      "filled_quantity": 0.025,
      "price": null,
      "average_price": 43250.50,
      "total": 1081.26,
      "fee": 0.54,
      "fee_asset": "USDT",
      "status": "filled",
      "created_at": "2025-11-17T10:30:00.000Z",
      "updated_at": "2025-11-17T10:30:00.000Z",
      "fills": [
        {
          "price": 43250.50,
          "quantity": 0.025,
          "timestamp": "2025-11-17T10:30:00.000Z"
        }
      ]
    }
  }
}
```

---

### 4.4 Cancel Order
**Endpoint:** `DELETE /orders/:orderId`

**Example:** `DELETE /orders/order-uuid-2`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Order cancelled successfully",
  "data": {
    "order": {
      "id": "order-uuid-2",
      "status": "cancelled",
      "updated_at": "2025-11-17T10:35:00.000Z"
    }
  }
}
```

**Error Response (400):**
```json
{
  "success": false,
  "error": "Order already filled, cannot cancel"
}
```

---

### 4.5 Cancel All Orders
**Endpoint:** `DELETE /orders/cancel-all`

**Query Parameters:**
- `symbol` - Cancel orders for specific symbol (optional)

**Example:** `DELETE /orders/cancel-all?symbol=BTCUSDT`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "All pending orders cancelled",
  "data": {
    "cancelled_count": 3,
    "order_ids": ["order-uuid-2", "order-uuid-3", "order-uuid-4"]
  }
}
```

---

## 5. Portfolio APIs

### 5.1 Get Portfolio Summary
**Endpoint:** `GET /portfolio/summary`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "total_value_usd": 15250.75,
    "total_pnl": 5250.75,
    "total_pnl_percentage": 52.51,
    "initial_balance": 10000.00,
    "assets": [
      {
        "asset": "BTC",
        "quantity": 0.20,
        "average_buy_price": 40000.00,
        "current_price": 43250.50,
        "total_value_usd": 8650.10,
        "pnl": 650.10,
        "pnl_percentage": 8.13,
        "allocation_percentage": 56.72
      },
      {
        "asset": "ETH",
        "quantity": 3.0,
        "average_buy_price": 2200.00,
        "current_price": 2280.75,
        "total_value_usd": 6842.25,
        "pnl": 242.25,
        "pnl_percentage": 3.67,
        "allocation_percentage": 44.86
      },
      {
        "asset": "USDT",
        "quantity": 8500.50,
        "average_buy_price": 1.00,
        "current_price": 1.00,
        "total_value_usd": 8500.50,
        "pnl": 0.00,
        "pnl_percentage": 0.00,
        "allocation_percentage": 55.74
      }
    ],
    "updated_at": "2025-11-17T10:30:00.000Z"
  }
}
```

---

### 5.2 Get Portfolio History
**Endpoint:** `GET /portfolio/history`

**Query Parameters:**
- `period` - Time period: 1d, 7d, 30d, 90d, 1y, all (default: 7d)
- `interval` - Data interval: 1h, 4h, 1d (default: 1h)

**Example:** `GET /portfolio/history?period=7d&interval=1h`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "period": "7d",
    "interval": "1h",
    "history": [
      {
        "timestamp": 1700136000000,
        "total_value_usd": 10000.00,
        "pnl": 0.00,
        "pnl_percentage": 0.00
      },
      {
        "timestamp": 1700139600000,
        "total_value_usd": 10150.25,
        "pnl": 150.25,
        "pnl_percentage": 1.50
      },
      {
        "timestamp": 1700222400000,
        "total_value_usd": 15250.75,
        "pnl": 5250.75,
        "pnl_percentage": 52.51
      }
    ]
  }
}
```

---

## 6. Trade History APIs

### 6.1 Get Trade History
**Endpoint:** `GET /trades/history`

**Query Parameters:**
- `symbol` - Filter by symbol (optional)
- `start_date` - Start date in ISO format (optional)
- `end_date` - End date in ISO format (optional)
- `limit` - Number of trades (default: 50)
- `offset` - Pagination offset (default: 0)

**Example:** `GET /trades/history?symbol=BTCUSDT&limit=20`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "trades": [
      {
        "id": "trade-uuid-1",
        "order_id": "order-uuid-1",
        "symbol": "BTCUSDT",
        "side": "buy",
        "price": 43250.50,
        "quantity": 0.025,
        "total": 1081.26,
        "fee": 0.54,
        "fee_asset": "USDT",
        "realized_pnl": 0.00,
        "timestamp": "2025-11-17T10:30:00.000Z"
      },
      {
        "id": "trade-uuid-2",
        "order_id": "order-uuid-5",
        "symbol": "ETHUSDT",
        "side": "sell",
        "price": 2280.75,
        "quantity": 1.0,
        "total": 2280.75,
        "fee": 1.14,
        "fee_asset": "USDT",
        "realized_pnl": 80.75,
        "timestamp": "2025-11-17T09:15:00.000Z"
      }
    ],
    "total": 2,
    "limit": 20,
    "offset": 0
  }
}
```

---

## 7. Watchlist APIs

### 7.1 Get Watchlist
**Endpoint:** `GET /watchlist`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "watchlist": [
      {
        "symbol": "BTCUSDT",
        "name": "Bitcoin",
        "price": 43250.50,
        "change_24h": 2.5,
        "volume_24h": 28500000000,
        "added_at": "2025-11-17T10:00:00.000Z"
      },
      {
        "symbol": "ETHUSDT",
        "name": "Ethereum",
        "price": 2280.75,
        "change_24h": -1.2,
        "volume_24h": 12000000000,
        "added_at": "2025-11-17T10:05:00.000Z"
      }
    ]
  }
}
```

---

### 7.2 Add to Watchlist
**Endpoint:** `POST /watchlist/add`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {access_token}"
}
```

**Request Body:**
```json
{
  "symbol": "BNBUSDT"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "message": "Symbol added to watchlist",
  "data": {
    "symbol": "BNBUSDT",
    "added_at": "2025-11-17T10:40:00.000Z"
  }
}
```

**Error Response (400):**
```json
{
  "success": false,
  "error": "Symbol already in watchlist"
}
```

---

### 7.3 Remove from Watchlist
**Endpoint:** `DELETE /watchlist/:symbol`

**Example:** `DELETE /watchlist/BNBUSDT`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Symbol removed from watchlist"
}
```

---

## 8. User Profile APIs

### 8.1 Get User Profile
**Endpoint:** `GET /user/profile`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user-uuid",
      "email": "user@example.com",
      "name": "John Doe",
      "role": "user",
      "avatar_url": null,
      "preferences": {
        "theme": "dark",
        "language": "en",
        "currency": "USD",
        "notifications": {
          "email": true,
          "push": false,
          "trade_alerts": true,
          "price_alerts": true
        }
      },
      "created_at": "2025-11-17T10:00:00.000Z",
      "updated_at": "2025-11-17T10:00:00.000Z"
    }
  }
}
```

---

### 8.2 Update User Profile
**Endpoint:** `PUT /user/profile`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {access_token}"
}
```

**Request Body:**
```json
{
  "name": "John Smith",
  "preferences": {
    "theme": "light",
    "currency": "EUR",
    "notifications": {
      "email": false,
      "push": true
    }
  }
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Profile updated successfully",
  "data": {
    "user": {
      "id": "user-uuid",
      "email": "user@example.com",
      "name": "John Smith",
      "preferences": {
        "theme": "light",
        "language": "en",
        "currency": "EUR",
        "notifications": {
          "email": false,
          "push": true,
          "trade_alerts": true,
          "price_alerts": true
        }
      },
      "updated_at": "2025-11-17T10:45:00.000Z"
    }
  }
}
```

---

### 8.3 Change Password
**Endpoint:** `POST /user/change-password`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {access_token}"
}
```

**Request Body:**
```json
{
  "current_password": "OldPassword123!",
  "new_password": "NewSecurePassword456!"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Password changed successfully"
}
```

**Error Response (400):**
```json
{
  "success": false,
  "error": "Current password is incorrect"
}
```

---

## 9. Admin APIs

### 9.1 Get All Users (Admin Only)
**Endpoint:** `GET /admin/users`

**Query Parameters:**
- `limit` - Number of users (default: 50)
- `offset` - Pagination offset (default: 0)
- `search` - Search by email or name (optional)

**Example:** `GET /admin/users?limit=20&search=john`

**Headers:**
```json
{
  "Authorization": "Bearer {admin_access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "user-uuid-1",
        "email": "user1@example.com",
        "name": "John Doe",
        "role": "user",
        "status": "active",
        "total_trades": 25,
        "total_volume": 50000.00,
        "created_at": "2025-11-17T10:00:00.000Z",
        "last_login": "2025-11-17T10:30:00.000Z"
      },
      {
        "id": "user-uuid-2",
        "email": "user2@example.com",
        "name": "Jane Smith",
        "role": "user",
        "status": "active",
        "total_trades": 10,
        "total_volume": 15000.00,
        "created_at": "2025-11-16T14:00:00.000Z",
        "last_login": "2025-11-17T09:15:00.000Z"
      }
    ],
    "total": 2,
    "limit": 20,
    "offset": 0
  }
}
```

---

### 9.2 Get User Details (Admin Only)
**Endpoint:** `GET /admin/users/:userId`

**Example:** `GET /admin/users/user-uuid-1`

**Headers:**
```json
{
  "Authorization": "Bearer {admin_access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "user": {
      "id": "user-uuid-1",
      "email": "user1@example.com",
      "name": "John Doe",
      "role": "user",
      "status": "active",
      "created_at": "2025-11-17T10:00:00.000Z",
      "last_login": "2025-11-17T10:30:00.000Z",
      "wallet": {
        "total_value_usd": 15250.75,
        "balances": [
          {
            "asset": "USDT",
            "total": 8500.50
          },
          {
            "asset": "BTC",
            "total": 0.20
          }
        ]
      },
      "statistics": {
        "total_trades": 25,
        "total_volume": 50000.00,
        "total_pnl": 5250.75,
        "win_rate": 68.5
      }
    }
  }
}
```

---

### 9.3 Update User Status (Admin Only)
**Endpoint:** `PUT /admin/users/:userId/status`

**Example:** `PUT /admin/users/user-uuid-1/status`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {admin_access_token}"
}
```

**Request Body:**
```json
{
  "status": "suspended"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "User status updated successfully",
  "data": {
    "user_id": "user-uuid-1",
    "status": "suspended",
    "updated_at": "2025-11-17T10:50:00.000Z"
  }
}
```

---

### 9.4 Update User Role (Admin Only)
**Endpoint:** `PUT /admin/users/:userId/role`

**Example:** `PUT /admin/users/user-uuid-1/role`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {admin_access_token}"
}
```

**Request Body:**
```json
{
  "role": "admin"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "User role updated successfully",
  "data": {
    "user_id": "user-uuid-1",
    "role": "admin",
    "updated_at": "2025-11-17T10:55:00.000Z"
  }
}
```

---

### 9.5 Delete User (Admin Only)
**Endpoint:** `DELETE /admin/users/:userId`

**Example:** `DELETE /admin/users/user-uuid-1`

**Headers:**
```json
{
  "Authorization": "Bearer {admin_access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "User deleted successfully"
}
```

---

### 9.6 Get Platform Statistics (Admin Only)
**Endpoint:** `GET /admin/statistics`

**Headers:**
```json
{
  "Authorization": "Bearer {admin_access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "total_users": 150,
    "active_users_24h": 45,
    "total_trades_24h": 320,
    "total_volume_24h": 1500000.00,
    "total_trades_all_time": 15000,
    "total_volume_all_time": 50000000.00,
    "top_trading_pairs": [
      {
        "symbol": "BTCUSDT",
        "volume_24h": 800000.00,
        "trades_24h": 150
      },
      {
        "symbol": "ETHUSDT",
        "volume_24h": 450000.00,
        "trades_24h": 95
      }
    ],
    "timestamp": "2025-11-17T10:30:00.000Z"
  }
}
```

---

## 10. Price Alert APIs

### 10.1 Create Price Alert
**Endpoint:** `POST /alerts/create`

**Headers:**
```json
{
  "Content-Type": "application/json",
  "Authorization": "Bearer {access_token}"
}
```

**Request Body:**
```json
{
  "symbol": "BTCUSDT",
  "condition": "above",
  "price": 45000.00,
  "notification_type": "email"
}
```

**Success Response (201):**
```json
{
  "success": true,
  "message": "Price alert created successfully",
  "data": {
    "alert": {
      "id": "alert-uuid-1",
      "user_id": "user-uuid",
      "symbol": "BTCUSDT",
      "condition": "above",
      "price": 45000.00,
      "notification_type": "email",
      "status": "active",
      "created_at": "2025-11-17T10:30:00.000Z"
    }
  }
}
```

---

### 10.2 Get Price Alerts
**Endpoint:** `GET /alerts`

**Query Parameters:**
- `status` - Filter by status: active, triggered, cancelled (optional)

**Example:** `GET /alerts?status=active`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "data": {
    "alerts": [
      {
        "id": "alert-uuid-1",
        "symbol": "BTCUSDT",
        "condition": "above",
        "price": 45000.00,
        "current_price": 43250.50,
        "notification_type": "email",
        "status": "active",
        "created_at": "2025-11-17T10:30:00.000Z"
      },
      {
        "id": "alert-uuid-2",
        "symbol": "ETHUSDT",
        "condition": "below",
        "price": 2200.00,
        "current_price": 2280.75,
        "notification_type": "push",
        "status": "active",
        "created_at": "2025-11-17T09:15:00.000Z"
      }
    ]
  }
}
```

---

### 10.3 Delete Price Alert
**Endpoint:** `DELETE /alerts/:alertId`

**Example:** `DELETE /alerts/alert-uuid-1`

**Headers:**
```json
{
  "Authorization": "Bearer {access_token}"
}
```

**Success Response (200):**
```json
{
  "success": true,
  "message": "Price alert deleted successfully"
}
```

---

## Error Codes

All API endpoints return standard HTTP status codes:

- `200` - Success
- `201` - Created successfully
- `400` - Bad request (validation error, insufficient balance, etc.)
- `401` - Unauthorized (invalid or missing token)
- `403` - Forbidden (insufficient permissions)
- `404` - Not found
- `409` - Conflict (duplicate entry)
- `500` - Internal server error

**Standard Error Response Format:**
```json
{
  "success": false,
  "error": "Error message description",
  "code": "ERROR_CODE",
  "timestamp": "2025-11-17T10:30:00.000Z"
}
```

---

## Rate Limiting

- Market data endpoints: 100 requests per minute
- Trading endpoints: 50 requests per minute
- Authentication endpoints: 10 requests per minute
- Admin endpoints: 200 requests per minute

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1700226060
```

---

## WebSocket API (Real-time Updates)

**WebSocket URL:**
```
wss://{projectId}.supabase.co/functions/v1/make-server-6b6c8494/ws
```

### Subscribe to Market Data
```json
{
  "action": "subscribe",
  "channel": "ticker",
  "symbol": "BTCUSDT"
}
```

**Server Response:**
```json
{
  "channel": "ticker",
  "symbol": "BTCUSDT",
  "data": {
    "price": 43250.50,
    "change_24h": 2.5,
    "volume_24h": 28500000000,
    "timestamp": 1700226000000
  }
}
```

### Subscribe to Order Book
```json
{
  "action": "subscribe",
  "channel": "orderbook",
  "symbol": "BTCUSDT"
}
```

### Subscribe to User Orders
```json
{
  "action": "subscribe",
  "channel": "user_orders",
  "token": "access_token"
}
```

---

## Notes

1. All timestamps are in Unix milliseconds or ISO 8601 format
2. All prices and amounts are in decimal format
3. This is a demo/educational platform with simulated data
4. For production use, implement proper security measures
5. All monetary values are returned as numbers, not strings
6. Symbols follow the format: BASE + QUOTE (e.g., BTCUSDT = BTC/USDT)
