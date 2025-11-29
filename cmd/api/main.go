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
		AllowOrigins:     []string{"http://localhost:3000"},
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

	db, err := data.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}

	marketRepo := repo.NewMarketRepo(db.DB)
	orderRepo  := repo.NewOrderRepo(db.DB)
	tradeRepo  := repo.NewTradeRepo(db.DB)
	walletRepo := repo.NewWalletRepo(db.DB)

	orderService := service.NewOrderService(db.DB, marketRepo, orderRepo, tradeRepo, walletRepo)

	handle := handler.NewHandler(orderService, marketRepo, orderRepo)

	routes.AuthRoutes(r, db)
	routes.WebSocketRoutes(r, handle)
	routes.MarketRoutes(r, handle)

	r.Use(middleware.RequireAuth(db.DB))
	
	routes.UserRoutes(r, db)
	routes.OrderRoutes(r, handle)

	defer db.Close()

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}