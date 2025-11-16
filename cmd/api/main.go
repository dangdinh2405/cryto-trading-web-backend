package main

import (
	"os"
	"log"

	"github.com/joho/godotenv"
	"github.com/gin-gonic/gin"

	"github.com/dangdinh2405/cryto-trading-web-backend/internal/data"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/route"
)

func main() {
	godotenv.Load()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	db, err := data.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}

	route.AuthRoutes(r, db)

	defer db.Close()

	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}