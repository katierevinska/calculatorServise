package store

import (
	"database/sql"
	"log"

	"github.com/katierevinska/calculatorService/internal"
)

type ExpressionStore struct {
	db *sql.DB
}

func NewExpressionStore(db *sql.DB) *ExpressionStore {
	return &ExpressionStore{db: db}
}

func (s *ExpressionStore) AddExpression(expr internal.Expression) error {
	var existingStatus string
	err := s.db.QueryRow("SELECT status FROM expressions WHERE id = ? AND user_id = ?", expr.ID, expr.UserID).Scan(&existingStatus)

	if err == sql.ErrNoRows {
		stmt, err := s.db.Prepare("INSERT INTO expressions (id, user_id, expression_string, status, result) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			log.Printf("Error preparing insert statement for expression: %v", err)
			return err
		}
		defer stmt.Close()
		_, err = stmt.Exec(expr.ID, expr.UserID, expr.ExpressionString, expr.Status, expr.Result)
		if err != nil {
			log.Printf("Error executing insert for expression %s: %v", expr.ID, err)
		}
		return err
	} else if err != nil {
		log.Printf("Error checking existence for expression %s: %v", expr.ID, err)
		return err
	}

	stmt, err := s.db.Prepare("UPDATE expressions SET status = ?, result = ?, expression_string = ? WHERE id = ? AND user_id = ?")
	if err != nil {
		log.Printf("Error preparing update statement for expression: %v", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(expr.Status, expr.Result, expr.ExpressionString, expr.ID, expr.UserID)
	if err != nil {
		log.Printf("Error executing update for expression %s: %v", expr.ID, err)
	}
	return err
}

func (s *ExpressionStore) UpdateExpressionStatusResult(expressionID, status, result string) error {
	stmt, err := s.db.Prepare("UPDATE expressions SET status = ?, result = ? WHERE id = ?")
	if err != nil {
		log.Printf("Error preparing update statement for expression status/result: %v", err)
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(status, result, expressionID)
	if err != nil {
		log.Printf("Error executing update for expression status/result %s: %v", expressionID, err)
		return err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("No expression found with ID %s to update status/result.", expressionID)
	}
	return nil
}

func (s *ExpressionStore) GetExpression(id string, userID int64) (internal.Expression, bool) {
	expr := internal.Expression{}
	err := s.db.QueryRow("SELECT id, user_id, expression_string, status, result, created_at FROM expressions WHERE id = ? AND user_id = ?", id, userID).
		Scan(&expr.ID, &expr.UserID, &expr.ExpressionString, &expr.Status, &expr.Result, &expr.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return internal.Expression{}, false
		}
		log.Printf("Error getting expression %s for user %d: %v", id, userID, err)
		return internal.Expression{}, false
	}
	return expr, true
}

func (s *ExpressionStore) GetAllExpressions(userID int64) []internal.Expression {
	rows, err := s.db.Query("SELECT id, user_id, expression_string, status, result, created_at FROM expressions WHERE user_id = ? ORDER BY created_at DESC", userID)
	if err != nil {
		log.Printf("Error getting all expressions for user %d: %v", userID, err)
		return []internal.Expression{}
	}
	defer rows.Close()

	expressionsList := []internal.Expression{}
	for rows.Next() {
		expr := internal.Expression{}
		err := rows.Scan(&expr.ID, &expr.UserID, &expr.ExpressionString, &expr.Status, &expr.Result, &expr.CreatedAt)
		if err != nil {
			log.Printf("Error scanning expression row for user %d: %v", userID, err)
			continue
		}
		expressionsList = append(expressionsList, expr)
	}
	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating expression rows for user %d: %v", userID, err)
	}
	return expressionsList
}
