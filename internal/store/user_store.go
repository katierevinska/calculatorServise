package store

import (
	"database/sql"
	"errors"

	"github.com/katierevinska/calculatorService/internal/models"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user with this login already exists")

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(login, password string) (int64, error) {
	hashedPassword, err := models.HashPassword(password)
	if err != nil {
		return 0, err
	}

	var existingID int64
	err = s.db.QueryRow("SELECT id FROM users WHERE login = ?", login).Scan(&existingID)
	if err == nil {
		return 0, ErrUserExists
	} else if err != sql.ErrNoRows {
		return 0, err
	}

	result, err := s.db.Exec("INSERT INTO users (login, password_hash) VALUES (?, ?)", login, hashedPassword)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *UserStore) GetUserByLogin(login string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow("SELECT id, login, password_hash FROM users WHERE login = ?", login).Scan(&user.ID, &user.Login, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
