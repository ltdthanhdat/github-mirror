package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dat-lt-amira/github-mirror/internal/models"
)

type PostgresUserStore struct {
	db *sql.DB
}

func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

func (s *PostgresUserStore) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	if err := s.db.QueryRow(`
		SELECT id, email, password_hash, full_name, is_admin, created_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.IsAdmin,
		&user.CreatedAt,
	); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	} else if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *PostgresUserStore) CreateUser(user *models.User) error {
	return s.db.QueryRow(`
		INSERT INTO users (email, password_hash, full_name, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, user.Email, user.PasswordHash, user.FullName, user.IsAdmin).Scan(&user.ID, &user.CreatedAt)
}

func (s *PostgresUserStore) EnsureAdminUser(ctx context.Context, email, password, fullName string) error {
	user := &models.User{
		Email:    email,
		FullName: fullName,
		IsAdmin:  true,
	}
	if err := user.SetPassword(password); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, full_name, is_admin)
		VALUES ($1, $2, $3, TRUE)
		ON CONFLICT (email)
		DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			is_admin = TRUE
	`, user.Email, user.PasswordHash, user.FullName)
	return err
}
