package route

import (
	"github.com/gin-gonic/gin"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/data"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/controller"
)

func AuthRoutes(r *gin.Engine, pg *data.Postgres) {
	auth := r.Group("/auth") 

	auth.POST("/register", controller.Register(pg.DB))
	auth.POST("/signin", controller.SignIn(pg.DB))
	auth.POST("/signout", controller.SignOut(pg.DB))
	auth.POST("/refresh", controller.RefreshToken(pg.DB))
}

// func UserRoutes(r *gin.Engine) {
// 	user := r.Group("/users") 

// 	user.GET("/me", handler.AuthMe())
// }