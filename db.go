package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Session struct {
	UserID   int64
	Host     string
	Port     int
	Username string
	Password []byte
}

func initDB(path string) {
	var err error
	db, err = sql.Open("sqlite3", path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			user_id    INTEGER PRIMARY KEY,
			host       TEXT    NOT NULL,
			port       INTEGER NOT NULL DEFAULT 22,
			username   TEXT    NOT NULL,
			password   BLOB    NOT NULL,
			last_used  TEXT    DEFAULT (datetime('now'))
		)
	`)
}

func saveSession(userID int64, host string, port int, username string, encPassword []byte) {
	db.Exec(`
		INSERT INTO sessions (user_id, host, port, username, password, last_used)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET
			host      = excluded.host,
			port      = excluded.port,
			username  = excluded.username,
			password  = excluded.password,
			last_used = datetime('now')
	`, userID, host, port, username, encPassword)
}

func getSession(userID int64) *Session {
	row := db.QueryRow("SELECT user_id, host, port, username, password FROM sessions WHERE user_id = ?", userID)
	var s Session
	if err := row.Scan(&s.UserID, &s.Host, &s.Port, &s.Username, &s.Password); err != nil {
		return nil
	}
	return &s
}

func deleteSession(userID int64) {
	db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
}
