package controller

import (
	"context"
	"database/sql"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	// "github.com/dangdinh2405/cryto-trading-web-backend/internal/models"
	// "github.com/dangdinh2405/cryto-trading-web-backend/internal/data"
)

type RegisterRequest struct {
	FirstName string		`json:"FirstName"`
    LastName  string		`json:"LastName"`
	Username  string 		`json:"Username"`
    Password  string 		`json:"Password"`
    Email     string		`json:"Email"`
	Phone     string		`json:"Phone"`
	Birthday  time.Time		`json:"Birthday"`
}

type SignInRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type jwtClaims struct {
	UserID string `json:"userId"`
	jwt.RegisteredClaims
}


var accessTTL = 15 * time.Minute
var refreshTTL = 7 * 24 * time.Hour


func Register(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid data"})
			return
		}

		if req.FirstName == "" || req.LastName == "" || req.Username == "" || req.Password == "" || req.Email == "" || req.Phone == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Frist Name, Last Name, Username, Password, Email, Phone and Brithday are required.",
			})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// 1. Check username hoặc email đã tồn tại chưa
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM users
				WHERE username = $1 OR email = $2 OR phone_number = $3
			)
		`, req.Username, req.Email, req.Phone).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (db check)"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"message": "username, phone or email already exists"})
			return
		}

		// 2. Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error encode password"})
			return
		}

		tx, err := db.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (tx)"})
			return
		}
		defer tx.Rollback()

		var userID uuid.UUID
		err = tx.QueryRowContext(ctx, `
			INSERT INTO users (
				first_name,
				last_name,
				username,
				email,
				phone_number,
				birthday,
				status
			) VALUES ($1, $2, $3, $4, $5, $6, 'active')
			RETURNING id
		`,
			req.FirstName,
			req.LastName,
			req.Username,
			req.Email,
			req.Phone,
			req.Birthday,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (insert user)"})
			return
		}

		now := time.Now()
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_auth (
				user_id,
				password_hash,
				last_password_change
			) VALUES ($1, $2, $3)
		`,
			userID,
			string(hashedPassword),
			now,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (insert auth)"})
			return
		}

		// 6. Commit
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (commit)"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}


func LogLoginActivity(db *sql.DB, userID uuid.UUID, ip, userAgent string, success bool) {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    _, err := db.ExecContext(ctx, `
        INSERT INTO login_activities (user_id, ip_address, user_agent, location, successful)
        VALUES ($1, $2, $3, $4, $5)
    `, userID, ip, userAgent, nil, success)
    if err != nil {
        log.Println("LogLoginActivity error:", err)
    }
}


func SignIn(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtSecret := os.Getenv("ACCESS_TOKEN_SECRET")

		var req SignInRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid data"})
			return
		}

		if req.Username == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "username, password are required."})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// 1. Lấy user + password_hash từ Postgres
		var (
			userID       uuid.UUID
			displayName  string
			passwordHash string
		)

		err := db.QueryRowContext(ctx, `
			SELECT 
				u.id,
				u.first_name || ' ' || u.last_name AS display_name,
				ua.password_hash
			FROM users u
			JOIN user_auth ua ON ua.user_id = u.id
			WHERE u.username = $1
		`, req.Username).Scan(&userID, &displayName, &passwordHash)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "Incorrect username or password"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (db)"})
			return
		}

		// 2. So sánh password
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Incorrect username or password"})
			return
		}

		// 3. Tạo JWT access token
		claims := jwtClaims{
			UserID: userID.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTTL)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		accessToken, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (jwt)"})
			return
		}

		// 4. Tạo refresh token random
		buf := make([]byte, 64)
		if _, err := rand.Read(buf); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Lỗi hệ thống (gen refresh token)"})
			return
		}
		refreshToken := hex.EncodeToString(buf)

		// 5. Hash refresh token trước khi lưu DB
		hashBytes := sha256.Sum256([]byte(refreshToken))
		tokenHash := hex.EncodeToString(hashBytes[:])

		expiresAt := time.Now().Add(refreshTTL)
		ipAddr := c.ClientIP()

		// 6. Lưu vào bảng refresh_tokens (đúng schema mới)
		_, err = db.ExecContext(ctx, `
			INSERT INTO refresh_tokens (
				user_id,
				token_hash,
				ip_address,
				expires_at
			) VALUES ($1, $2, $3::inet, $4)
		`,
			userID,
			tokenHash,
			ipAddr,
			expiresAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "System error (create refresh token in postgres)"})
			return
		}

		// 7. Set cookie refreshToken
		// c.SetSameSite(http.SameSiteNoneMode)
		secure := false // đổi true khi dùng HTTPS

		c.SetCookie(
			"refreshToken",
			refreshToken,              // raw token chỉ nằm ở cookie
			int(refreshTTL.Seconds()), // max-age
			"/",
			"",
			secure,
			true, // httpOnly
		)
		
		LogLoginActivity(db, userID, ipAddr, c.Request.UserAgent(), true)

		c.JSON(http.StatusOK, gin.H{
			"message":     "User " + displayName + " đã logged in!",
			"accessToken": accessToken,
			"expiresIn":   int(accessTTL.Seconds()),
		})
	}
}

func SignOut(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		refreshToken, err := c.Cookie("refreshToken")
		if err == nil && refreshToken != "" {

			hashBytes := sha256.Sum256([]byte(refreshToken))
			tokenHash := hex.EncodeToString(hashBytes[:])

			ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
			defer cancel()

			// Soft delete: cập nhật revoked_at
			_, err := db.ExecContext(ctx, `
				UPDATE refresh_tokens
				SET revoked_at = NOW()
				WHERE token_hash = $1
				  AND revoked_at IS NULL
			`, tokenHash)

			// Không xử lý lỗi (logout vẫn tiếp tục)
			_ = err

			// Xoá cookie
			// c.SetSameSite(http.SameSiteNoneMode)
			secure := false // dùng HTTPS thì đổi true

			c.SetCookie(
				"refreshToken",
				"",   // xoá value
				-1,   // MaxAge < 0 => trình duyệt xoá cookie
				"/",
				"",
				secure,
				true,
			)
		}

		c.Status(http.StatusNoContent)
	}
}

func RefreshToken(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtSecret := os.Getenv("ACCESS_TOKEN_SECRET")

		// 1) Lấy refresh token từ cookie
		refreshToken, err := c.Cookie("refreshToken")
		if err != nil || refreshToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Token không tồn tại."})
			return
		}

		// 2) Hash refresh token để so với token_hash trong DB
		hashBytes := sha256.Sum256([]byte(refreshToken))
		tokenHash := hex.EncodeToString(hashBytes[:])

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// 3) Tìm bản ghi refresh token trong DB
		var (
			userID    uuid.UUID
			expiresAt time.Time
			revokedAt *time.Time
		)

		err = db.QueryRowContext(ctx, `
			SELECT user_id, expires_at, revoked_at
			FROM refresh_tokens
			WHERE token_hash = $1
		`, tokenHash).Scan(&userID, &expiresAt, &revokedAt)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusForbidden, gin.H{"message": "Token không hợp lệ hoặc đã hết hạn"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Lỗi hệ thống (db)"})
			return
		}

		// 4) Kiểm tra token đã bị revoke chưa
		if revokedAt != nil {
			c.JSON(http.StatusForbidden, gin.H{"message": "Token đã bị thu hồi."})
			return
		}

		// 5) Kiểm tra hết hạn
		if time.Now().After(expiresAt) {
			// Có thể xoá luôn bản ghi hết hạn nếu bạn muốn
			// go func() {
			// 	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
			// 	defer cancel2()
			// 	_, _ = db.ExecContext(ctx2, `
			// 		DELETE FROM refresh_tokens
			// 		WHERE token_hash = $1
			// 	`, tokenHash)
			// }()
			c.JSON(http.StatusForbidden, gin.H{"message": "Token đã hết hạn."})
			return
		}

		// 6) Tạo access token mới
		claims := jwtClaims{
			UserID: userID.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTTL)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		accessToken, err := newToken.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Lỗi hệ thống (jwt)"})
			return
		}

		// 7) Trả về access token mới
		c.JSON(http.StatusOK, gin.H{
			"accessToken": accessToken,
			"expiresIn":   int(accessTTL.Seconds()),
		})
	}
}
