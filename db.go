package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

// Create database connection
func openDB(url string) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", url)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	return
}
