package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dataSourceName string) (*sql.DB, error) {
	dbDir := filepath.Dir(dataSourceName)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Printf("Error creating database directory: %v", err)
			return nil, err
		}
		log.Printf("Created database directory: %s", dbDir)
	}

	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	if err = createTables(db); err != nil {
		return nil, err
	}
	log.Println("Database initialized and tables created successfully.")
	return db, nil
}

func createTables(db *sql.DB) error {
	createUserTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		login TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL
	);`

	_, err := db.Exec(createUserTableSQL)
	if err != nil {
		log.Printf("Error creating users table: %v", err)
		return err
	}

	createExpressionsTableSQL := `
	CREATE TABLE IF NOT EXISTS expressions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expression_string TEXT NOT NULL, -- Добавим поле для хранения исходного выражения
		status TEXT NOT NULL,
		result TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	_, err = db.Exec(createExpressionsTableSQL)
	if err != nil {
		log.Printf("Error creating expressions table: %v", err)
		return err
	}
	return nil
}
