package auth

import (
	"errors"

	"github.com/dat-lt-amira/github-mirror/internal/models"
)

// UserStore defines the methods required for user storage.
type UserStore interface {
	GetUserByEmail(email string) (*models.User, error)
	CreateUser(user *models.User) error
}

// ErrUserNotFound is returned when a user with the given email is not found.
var ErrUserNotFound = errors.New("user not found")

// InMemoryUserStore is a simple in-memory implementation of UserStore for development.
type InMemoryUserStore struct {
	users  map[string]*models.User
	nextID uint64
}

// NewInMemoryUserStore creates a new InMemoryUserStore with the given users.
func NewInMemoryUserStore(users []*models.User) *InMemoryUserStore {
	store := &InMemoryUserStore{
		users:  make(map[string]*models.User),
		nextID: 1,
	}
	for _, u := range users {
		if u.ID == 0 {
			u.ID = store.nextID
			store.nextID++
		} else if u.ID >= store.nextID {
			store.nextID = u.ID + 1
		}
		store.users[u.Email] = u
	}
	return store
}

// GetUserByEmail returns the user with the given email.
func (s *InMemoryUserStore) GetUserByEmail(email string) (*models.User, error) {
	if u, ok := s.users[email]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

// CreateUser creates a new user in the store.
func (s *InMemoryUserStore) CreateUser(user *models.User) error {
	if _, ok := s.users[user.Email]; ok {
		return errors.New("user already exists")
	}
	if user.ID == 0 {
		user.ID = s.nextID
		s.nextID++
	}
	s.users[user.Email] = user
	return nil
}
