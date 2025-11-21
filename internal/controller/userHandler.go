package controller

import (
	"net/http"
	"database/sql"
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/middleware"
)

type LoginActivity struct {
    ID        string     `json:"id"        db:"id"`
    UserID    *string    `json:"userId,omitempty" db:"user_id"`   
    IP        string     `json:"ip"        db:"ip_address"`         
    UserAgent string     `json:"userAgent" db:"user_agent"`
    Location  *string    `json:"location,omitempty" db:"location"`   
    Success   bool       `json:"success"   db:"successful"`         
    CreatedAt time.Time  `json:"createdAt" db:"created_at"`
}


func AuthMe() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy user từ context (đã được set bởi middleware)
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "User not found in context"})
			return
		}

		// Convert user từ context về UserContext type
		userInfo, ok := user.(middleware.UserContext)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Invalid user data"})
			return
		}

		// Trả về thông tin user
		c.JSON(http.StatusOK, gin.H{
			"user": userInfo,
		})
	}
}

func GetLoginActivityHandler(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1) Lấy user từ context (middleware đã set "user")
        userVal, ok := c.Get("user")
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        // Tùy middleware: thường là AuthUser hoặc *AuthUser
        var userID uuid.UUID

        if u, ok := userVal.(middleware.UserContext); ok {
            userID = u.ID
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data in context"})
            return
        }

        if userID == uuid.Nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Empty user id"})
            return
        }

        // 2) Query DB
        ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
        defer cancel()

        rows, err := db.QueryContext(ctx, `
            SELECT id, user_id, ip_address, user_agent, location, successful, created_at
            FROM login_activities
            WHERE user_id = $1::uuid
            ORDER BY created_at DESC
            LIMIT 50
        `, userID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
            return
        }
        defer rows.Close()

        var activities []LoginActivity

        for rows.Next() {
            var a LoginActivity
            if err := rows.Scan(
                &a.ID,
                &a.UserID,
                &a.IP,
                &a.UserAgent,
                &a.Location,
                &a.Success,
                &a.CreatedAt,
            ); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan error"})
                return
            }
            activities = append(activities, a)
        }

        if err := rows.Err(); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Rows error"})
            return
        }

        // 3) Trả về list login activities
        // Frontend: response.data = []LoginActivity
        c.JSON(http.StatusOK, activities)
    }
}

func GetUserBalance(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        userVal, ok := c.Get("user")
        if !ok {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            return
        }

        var userID uuid.UUID

        if u, ok := userVal.(middleware.UserContext); ok {
            userID = u.ID
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data in context"})
            return
        }

        if userID == uuid.Nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Empty user id"})
            return
        }

        ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
        defer cancel()


        rows, err := db.QueryContext(ctx,`
            SELECT a.symbol, w.balance, w.in_orders
            FROM wallets w
            JOIN assets a ON a.id = w.asset_id
            WHERE w.user_id = $1
        `, userID)

        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"message": "Lỗi Querry database error"})
            return
        }
        defer rows.Close()

        // map[string]any để trả về JSON đúng format
        balances := make(map[string]map[string]any)

        for rows.Next() {
            var symbol string
            var balance, inOrders float64

            if err := rows.Scan(&symbol, &balance, &inOrders); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"message": "Read data error"})
                return
            }

            balances[symbol] = map[string]any{
                "available": balance,
                "inOrders":  inOrders,
            }
        }

        c.JSON(http.StatusOK, balances)
    }
}
