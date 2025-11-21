package middleware

import (
	"context"
	"net/http"
	"database/sql"
	"strings"
	"time"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserContext struct {
	ID             uuid.UUID  `json:"id"`
	FirstName      string     `json:"first_name"`
	LastName       string     `json:"last_name"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Phone          string    `json:"phone_number"`
	Birthday       time.Time `json:"birthday"`
	AvatarURL      *string    `json:"avatar_url"`
	Passkey		   bool    	  `json:"passkey_enabled"`
	Status    	   string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`

}

func RequireAuth(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtSecret := os.Getenv("ACCESS_TOKEN_SECRET")

		// 1. Lấy Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Access token not found"})
			c.Abort()
			return
		}

		// 2. Cắt "Bearer " lấy token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 3. Parse + verify token
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// Optional: kiểm tra signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusForbidden, gin.H{"message": "Access token expired or incorrect"})
			c.Abort()
			return
		}

		// 4. Lấy claims (userId) từ token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
			c.Abort()
			return
		}

		userIDStr, ok := claims["userId"].(string)
		if !ok || userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "userId not found in token"})
			c.Abort()
			return
		}

		// 5. Parse UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid user id"})
			c.Abort()
			return
		}

		// 6. Query user từ Postgres
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var (
			id             uuid.UUID  
			first_name      string     
			last_name       string     
			username       string     
			email          string     
			phone_number          string    
			birthday       time.Time 
			avatar_url      *string    
			passkey_enabled		   bool    	  
			status    	   string         
			created_at      time.Time
			updated_at	   time.Time
		)

		err = db.QueryRowContext(ctx, `
			SELECT *
			FROM users
			WHERE id = $1
			  AND status = 'active'
		`, userID).Scan(&id, &first_name, &last_name, &username, &email, &phone_number, &birthday, &avatar_url, &passkey_enabled, &status, &created_at, &updated_at)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"message": "User does not exist"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (db)"})
			}
			c.Abort()
			return
		}

		// 7. Gắn user vào context
		userCtx := UserContext{
			ID         : id,
			FirstName  : first_name,
			LastName   : last_name,
			Username   : username,
			Email      : email,
			Phone      : phone_number,
			Birthday   : birthday,
			AvatarURL  : avatar_url,
			Passkey	   : passkey_enabled,
			Status     : status,
			CreatedAt  : created_at,
			UpdatedAt  : updated_at,
		}

		c.Set("user", userCtx)
		c.Next()
	}
}