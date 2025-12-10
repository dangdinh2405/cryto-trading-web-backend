# Crypto Trading Platform Backend

Backend API cho nền tảng giao dịch tiền mã hóa, xây dựng bằng **Go** với **Gin framework**.

## 🛠 Tech Stack

| Thành phần | Công nghệ |
|------------|-----------|
| **Language** | Go 1.25 |
| **Framework** | Gin |
| **Database** | PostgreSQL |
| **Cache** | Redis |
| **WebSocket** | Gorilla WebSocket |
| **Auth** | JWT (golang-jwt/v5) |
| **Container** | Docker |

## 📁 Cấu trúc dự án

```
├── cmd/api/main.go      # Entry point
├── internal/
│   ├── controller/      # Auth & User controllers
│   ├── handler/         # HTTP & WebSocket handlers
│   ├── middleware/      # JWT authentication
│   ├── models/          # Data models
│   ├── repo/            # Database repositories
│   ├── routes/          # API route definitions
│   ├── service/         # Business logic (Order matching)
│   └── data/            # PostgreSQL & Redis connections
└── docker-compose.yml
```

## ✨ Tính năng chính

### 🔐 Authentication
- Đăng ký / Đăng nhập với email & password
- JWT access token & refresh token
- Session management

### 💰 Order Management
- **Order Types**: Market, Limit
- **Time-in-Force**: GTC, IOC, FOK, POST_ONLY
- **Actions**: Place, Cancel, Amend orders
- **Matching Engine**: Tự động khớp lệnh buy/sell

### 📊 Market Data
- Danh sách markets (pairs)
- OHLCV candlestick data
- Order book (bids/asks)

### 📡 Real-time WebSocket
| Endpoint | Mô tả |
|----------|-------|
| `/ws/market-prices` | Live candle updates (OHLCV) |
| `/ws/orderbook` | Real-time order book |

Subscribe theo symbol:
```json
{"type": "subscribe", "symbols": ["BTCUSDT", "ETHUSDT"]}
```

### 💼 Wallet
- Xem số dư (available/locked)
- Lịch sử giao dịch
- Tự động cập nhật khi khớp lệnh

## 🚀 Chạy dự án

### Prerequisites
- Go 1.25+
- PostgreSQL
- Redis (optional)

### Environment Variables
```env
PORT=10000
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=yourpassword
POSTGRES_DB_NAME=crypto_trading
ACCESS_TOKEN_SECRET=your-jwt-secret
REDIS_HOST=localhost:6379
```

### Run locally
```bash
go mod download
go run ./cmd/api
```

### Docker
```bash
docker-compose up --build
```

## 📚 API Endpoints

### Authentication
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| POST | `/auth/register` | Đăng ký |
| POST | `/auth/login` | Đăng nhập |
| POST | `/auth/logout` | Đăng xuất |
| POST | `/auth/refresh` | Refresh token |

### User (🔒 Auth Required)
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/user/profile` | Thông tin user |
| GET | `/user/balance` | Số dư ví |
| GET | `/user/trades` | Lịch sử trades |
| GET | `/user/login-activity` | Lịch sử đăng nhập |

### Market
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/market/list` | Danh sách markets |
| GET | `/market/candles` | OHLCV data |

### Orders (🔒 Auth Required)
| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/orders` | Danh sách orders |
| POST | `/orders` | Đặt lệnh |
| DELETE | `/orders/:id` | Hủy lệnh |
| PUT | `/orders/:id` | Sửa lệnh |

## 🔄 Order Flow

```
1. User đặt lệnh → lockFunds (khóa số dư)
2. Matching Engine tìm lệnh đối ứng
3. Khớp lệnh → tạo Trade → settle (chuyển tiền)
4. Cập nhật Order status & Wallet balance
5. Broadcast qua WebSocket
```