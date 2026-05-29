package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system.
type User struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	PasswordHash string  `json:"-"` // never serialize the hash
	FullName  string    `json:"full_name,omitempty"`
	IsAdmin   bool      `json:"is_admin,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SetPassword hashes the password and stores it in the PasswordHash field.
func (u *User) SetPassword(password string) error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedBytes)
	return nil
}

// CheckPassword compares the hashed password with the provided password.
func (u *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
}