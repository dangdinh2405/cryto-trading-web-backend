package models

import (
    "time"

    "github.com/google/uuid"
)

type User struct {
    ID             uuid.UUID  `db:"id" json:"id"`
	FirstName      string     `db:"first_name" json:"first_name"`
	LastName       string     `db:"last_name" json:"last_name"`
    Username       string     `db:"username" json:"username"`
	Email          string     `db:"email" json:"email"`
	Phone          *string    `db:"phone_number" json:"phone_number"`
	Brithday       *time.Time `db:"birthday" json:"birthday"`
	AvatarURL      *string    `db:"avatar_url" json:"avatar_url"`
    Passkey		   bool    	  `db:"passkey_enabled" json:"passkey_enabled"`
    Status    	   string     `db:"status" json:"status"`
    CreatedAt      time.Time  `db:"created_at" json:"createdAt"`
    UpdatedAt      time.Time  `db:"updated_at" json:"updatedAt"`
}

type UserAuth struct {
    UserID         uuid.UUID  `db:"user_id" json:"user_id"`
	PasswordHash   string     `db:"password_hash" json:"password_hash"`
	PasskeyPublic  *string    `db:"passkey_public_key" json:"passkey_public_key"`
    TwoFASecret    *string    `db:"twofa_secret" json:"twofa_secret"`
	TwoFA          bool       `db:"twofa_enabled" json:"twofa_enabled"`
	LastPassword   *time.Time `db:"last_password_change" json:"last_password_change"`
}
