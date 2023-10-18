package database

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func GetConnection() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", "linkshortener.db")

	if err != nil {
		return nil, err
	}

	return db, nil
}
