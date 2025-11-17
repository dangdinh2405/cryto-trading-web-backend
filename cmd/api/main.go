package main

import (
	"os"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/data"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/route"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/middleware"
	
)

func main() {
	godotenv.Load()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
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

	route.AuthRoutes(r, db)

	r.Use(middleware.RequireAuth(db.DB))
	route.UserRoutes(r)


	defer db.Close()

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}