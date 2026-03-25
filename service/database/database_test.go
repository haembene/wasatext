package database

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewDatabase(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite3 memory database: %v", err)
	}
	defer dbConn.Close()

	appDB, err := New(dbConn)
	if err != nil {
		t.Fatalf("Failed to initialize AppDatabase: %v", err)
	}

	if err := appDB.Ping(); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestLoginOrCreateUser(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite3 memory database: %v", err)
	}
	defer dbConn.Close()

	appDB, err := New(dbConn)
	if err != nil {
		t.Fatalf("Failed to initialize AppDatabase: %v", err)
	}

	userID, err := appDB.LoginOrCreateUser("testuser")
	if err != nil {
		t.Fatalf("LoginOrCreateUser failed: %v", err)
	}
	if userID == "" {
		t.Errorf("Expected valid user ID, got empty string")
	}

	// Logging in again with the same username should return the identical ID
	userID2, err := appDB.LoginOrCreateUser("testuser")
	if err != nil {
		t.Fatalf("LoginOrCreateUser (existing) failed: %v", err)
	}
	if userID != userID2 {
		t.Errorf("Expected same user ID %q, got %q", userID, userID2)
	}
}
