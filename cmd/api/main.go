package main

import (
	"os"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/data"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/routes"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/repo"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/handler"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/service"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/middleware"
)

func main() {
	godotenv.Load()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3001"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Upgrade", "Connection"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	// Initialize PostgreSQL
	db, err := data.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize Redis
	redis, err := data.NewRedis()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v. Proceeding without cache.", err)
		redis = nil // Continue without Redis
	}
	if redis != nil {
		defer redis.Close()
	}

	// Create cache service
	var cacheService *service.CacheService
	if redis != nil {
		cacheService = service.NewCacheService(redis.Client)
		log.Println("Cache service initialized with Redis")
	} else {
		log.Println("Cache service disabled (Redis unavailable)")
	}

	// Initialize repositories
	marketRepo := repo.NewMarketRepo(db.DB)
	orderRepo  := repo.NewOrderRepo(db.DB)
	tradeRepo  := repo.NewTradeRepo(db.DB)
	walletRepo := repo.NewWalletRepo(db.DB)

	// Initialize services with cache
	orderService := service.NewOrderService(db.DB, marketRepo, orderRepo, tradeRepo, walletRepo, cacheService)

	// Initialize handlers with cache
	handle := handler.NewHandler(orderService, marketRepo, orderRepo, cacheService)

	// Setup routes
	routes.HealthRoutes(r) // Public health check for ECS/Docker
	routes.AuthRoutes(r, db)
	routes.WebSocketRoutes(r, handle)
	routes.MarketRoutes(r, handle)

	r.Use(middleware.RequireAuth(db.DB))
	
	routes.UserRoutes(r, db)
	routes.OrderRoutes(r, handle)

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}