package store_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/katierevinska/calculatorService/internal/database"
	"github.com/katierevinska/calculatorService/internal/models"
	"github.com/katierevinska/calculatorService/internal/store"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserStoreTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	db, err := database.InitDB(":memory:")
	require.NoError(t, err)

	return db, func() {
		db.Close()
	}
}

func TestUserStore(t *testing.T) {
	db, teardown := setupUserStoreTestDB(t)
	defer teardown()

	userStore := store.NewUserStore(db)

	login := "testuser"
	password := "password123"

	t.Run("CreateUser_Success", func(t *testing.T) {
		id, err := userStore.CreateUser(login, password)
		require.NoError(t, err)
		assert.Greater(t, id, int64(0))
	})

	t.Run("CreateUser_UserExists", func(t *testing.T) {
		_, err := userStore.CreateUser(login, "anotherpassword")
		require.Error(t, err)
		assert.True(t, errors.Is(err, store.ErrUserExists))
	})

	t.Run("GetUserByLogin_Success", func(t *testing.T) {
		user, err := userStore.GetUserByLogin(login)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, login, user.Login)
		assert.True(t, models.CheckPasswordHash(password, user.PasswordHash))
	})

	t.Run("GetUserByLogin_NotFound", func(t *testing.T) {
		_, err := userStore.GetUserByLogin("nonexistentuser")
		require.Error(t, err)
		assert.True(t, errors.Is(err, store.ErrUserNotFound))
	})
}
