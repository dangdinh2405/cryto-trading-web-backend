package route

import (
	"github.com/gin-gonic/gin"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/data"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/controller"
)

func AuthRoutes(r *gin.Engine, pg *data.Postgres) {
	auth := r.Group("/auth") 

	auth.POST("/register", controller.Register(pg.DB))
	auth.POST("/login", controller.SignIn(pg.DB))
	auth.POST("/logout", controller.SignOut(pg.DB))
	auth.POST("/refresh", controller.RefreshToken(pg.DB))
}

func UserRoutes(r *gin.Engine, pg *data.Postgres) {
	user := r.Group("/user") 

	user.GET("/profile", controller.AuthMe())	
	user.GET("/login-activity", controller.GetLoginActivityHandler(pg.DB))
	user.GET("/balance", controller.GetUserBalance(pg.DB))
}