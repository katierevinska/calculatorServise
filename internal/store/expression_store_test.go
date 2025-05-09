package store_test

import (
	"database/sql"
	"testing"

	"github.com/katierevinska/calculatorService/internal"
	"github.com/katierevinska/calculatorService/internal/database"
	"github.com/katierevinska/calculatorService/internal/store"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupExpressionStoreTestDB(t *testing.T) (*sql.DB, int64, func()) {
	t.Helper()
	db, err := database.InitDB(":memory:")
	require.NoError(t, err)

	userStore := store.NewUserStore(db)
	userID, err := userStore.CreateUser("expruser", "password")
	require.NoError(t, err)

	return db, userID, func() {
		db.Close()
	}
}

func TestExpressionStore(t *testing.T) {
	db, userID, teardown := setupExpressionStoreTestDB(t)
	defer teardown()

	exprStore := store.NewExpressionStore(db)

	expr1 := internal.Expression{
		ID:               "expr-id-1",
		UserID:           userID,
		ExpressionString: "1+2",
		Status:           "in progress",
	}

	t.Run("AddExpression_New", func(t *testing.T) {
		err := exprStore.AddExpression(expr1)
		require.NoError(t, err)

		retrieved, exists := exprStore.GetExpression(expr1.ID, userID)
		require.True(t, exists)
		assert.Equal(t, expr1.ExpressionString, retrieved.ExpressionString)
		assert.Equal(t, expr1.Status, retrieved.Status)
		assert.NotEmpty(t, retrieved.CreatedAt)
	})

	t.Run("AddExpression_UpdateExisting", func(t *testing.T) {
		updatedExpr1 := expr1
		updatedExpr1.Status = "calculated"
		updatedExpr1.Result = "3.0000000000"
		updatedExpr1.ExpressionString = "1+2"

		err := exprStore.AddExpression(updatedExpr1)
		require.NoError(t, err)

		retrieved, exists := exprStore.GetExpression(expr1.ID, userID)
		require.True(t, exists)
		assert.Equal(t, updatedExpr1.Status, retrieved.Status)
		assert.Equal(t, updatedExpr1.Result, retrieved.Result)
	})

	t.Run("UpdateExpressionStatusResult", func(t *testing.T) {
		err := exprStore.UpdateExpressionStatusResult(expr1.ID, "error", "division by zero")
		require.NoError(t, err)

		retrieved, exists := exprStore.GetExpression(expr1.ID, userID)
		require.True(t, exists)
		assert.Equal(t, "error", retrieved.Status)
		assert.Equal(t, "division by zero", retrieved.Result)
	})

	t.Run("GetAllExpressions", func(t *testing.T) {
		expr2 := internal.Expression{
			ID:               "expr-id-2",
			UserID:           userID,
			ExpressionString: "3*4",
			Status:           "in progress",
		}
		err := exprStore.AddExpression(expr2)
		require.NoError(t, err)

		userStore := store.NewUserStore(db)
		otherUserID, _ := userStore.CreateUser("otheruser", "pass")
		exprOtherUser := internal.Expression{ID: "expr-other", UserID: otherUserID, ExpressionString: "9-1", Status: "calculated", Result: "8"}
		err = exprStore.AddExpression(exprOtherUser)
		require.NoError(t, err)

		allExprs := exprStore.GetAllExpressions(userID)
		assert.Len(t, allExprs, 2, "Should only get expressions for the given userID")
		ids := make(map[string]bool)
		for _, e := range allExprs {
			ids[e.ID] = true
			assert.Equal(t, userID, e.UserID)
		}
		assert.True(t, ids[expr1.ID])
		assert.True(t, ids[expr2.ID])

		allExprsOtherUser := exprStore.GetAllExpressions(otherUserID)
		assert.Len(t, allExprsOtherUser, 1)
		assert.Equal(t, "expr-other", allExprsOtherUser[0].ID)
	})

	t.Run("GetExpression_NotFound", func(t *testing.T) {
		_, exists := exprStore.GetExpression("non-existent-id", userID)
		assert.False(t, exists)
	})

	t.Run("GetExpression_WrongUser", func(t *testing.T) {
		_, exists := exprStore.GetExpression(expr1.ID, userID+99)
		assert.False(t, exists)
	})
}
