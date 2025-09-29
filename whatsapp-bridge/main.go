package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func sqliteDSN(path string) string {
	// mode=rwc: read/write & create; busy_timeout to avoid transient locks
	return fmt.Sprintf(
		"file:%s?mode=rwc&_foreign_keys=on&_busy_timeout=30000",
		filepath.ToSlash(path),
	)
}

func absBasePath(base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("WA_DB_PATH not set")
	}
	if !filepath.IsAbs(base) {
		base = filepath.Join("/app/whatsapp-bridge", base)
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", base, err)
	}
	return base, nil
}

func initSQLite(path string) error {
	db, err := sql.Open("sqlite3", sqliteDSN(path))
	if err != nil {
		return err
	}
	defer db.Close()

	// enable WAL + set busy timeout
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return fmt.Errorf("set WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=30000;"); err != nil {
		return fmt.Errorf("set busy_timeout: %w", err)
	}

	// smoke write (ensures dir + WAL files are writable)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS __ping (x INTEGER);
	                       INSERT INTO __ping (x) VALUES (1);`); err != nil {
		return fmt.Errorf("smoke write: %w", err)
	}
	return db.Ping()
}

func main() {
	baseEnv := strings.TrimSpace(os.Getenv("WA_DB_PATH"))
	base, err := absBasePath(baseEnv)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		os.Exit(1)
	}

	sessionDB := filepath.Join(base, "whatsapp.db")  // whatsmeow device/session
	messagesDB := filepath.Join(base, "messages.db") // your message log

	fmt.Printf("Init DBs at base=%s\n", base)
	fmt.Printf(" session=%s\n messages=%s\n", sessionDB, messagesDB)

	if err := initSQLite(sessionDB); err != nil {
		fmt.Printf("ERR init session db: %v\n", err)
		os.Exit(2)
	}
	if err := initSQLite(messagesDB); err != nil {
		fmt.Printf("ERR init messages db: %v\n", err)
		os.Exit(3)
	}

	fmt.Println("OK: created/opened both DBs and verified writes (WAL enabled).")
}
