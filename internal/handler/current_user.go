package handler

import (
	"errors"	
	"time"

	"github.com/gin-gonic/gin"
	"github.com/dangdinh2405/cryto-trading-web-backend/internal/middleware"
	"github.com/google/uuid"
)

type CurrentUser struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	Birthday  time.Time `json:"birthday"`
	AvatarURL *string    `json:"avatar_url"`
	Passkey   bool      `json:"passkey_enabled"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func GetCurrentUser(c *gin.Context) (*CurrentUser, error) {
    v, ok := c.Get("user")
    if !ok || v == nil {
        return nil, errors.New("user not found in context")
    }

    if uc, ok := v.(middleware.UserContext); ok {
        return &CurrentUser{
            ID:        uc.ID,
            FirstName: uc.FirstName,
            LastName:  uc.LastName,
            Username:  uc.Username,
            Email:     uc.Email,
            Phone:     uc.Phone,
            Birthday:  uc.Birthday,  
            AvatarURL: uc.AvatarURL,
            Passkey:   uc.Passkey,
            Status:    uc.Status,
            CreatedAt: uc.CreatedAt,
            UpdatedAt: uc.UpdatedAt,
        }, nil
    }

    if u, ok := v.(*CurrentUser); ok {
        return u, nil
    }

    return nil, errors.New("invalid user type in context")
}

