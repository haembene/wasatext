package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// AppDatabase is the high level interface for the DB
type AppDatabase interface {
	// Login
	LoginOrCreateUser(name string) (string, error)

	// Users
	FindUsers(prefix string) ([]User, error)
	IsUserNameTaken(name string) (bool, error)
	UpdateUserName(userID, newName string) error
	UpdateUserPhoto(userID, url string) error

	// Conversations
	GetMyConversations(userID string, usernameFilter *string) ([]ConversationSnippet, error)
	GetConversation(meID, conversationID string) ([]Message, error)
	CreateConversation(myID, otherID string) (ConversationSnippet, error)
	SetConversationRead(myID, otherID string) error

	// Messages
	SendMessage(senderID, recipientID, content string, replyTo *string) (Message, error)
	ForwardMessage(messageID, newRecipientID, senderID string) (Message, error)
	DeleteMessage(messageID, userID string) error
	AddReaction(messageID, userID, emoticon string) error
	RemoveReaction(messageID, userID, emoticon string) error

	// Groups
	AddUserToGroup(groupID, userIDToAdd, requesterID string) error
	LeaveGroup(groupID, userID, requesterID string) error
	SetGroupName(groupID, requesterID, newName string) error
	SetGroupPhoto(groupID, requesterID, newPhotoURL string) error

	// Health check
	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building an AppDatabase")
	}

	// Check if `users` table exists — this tells us if the DB is initialized
	var tableName string
	err := db.QueryRow(`
		SELECT name 
		FROM sqlite_master 
		WHERE type='table' AND name='users';
	`).Scan(&tableName)

	if errors.Is(err, sql.ErrNoRows) {

		// The database is empty → create all tables
		schema := `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			photoUrl TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			senderId TEXT NOT NULL,
			recipientId TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			content TEXT NOT NULL,
			isPhoto INTEGER NOT NULL,
			status INTEGER NOT NULL,
			replyTo TEXT
		);

		CREATE TABLE IF NOT EXISTS groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			photoUrl TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS group_members (
			groupId TEXT NOT NULL,
			userId TEXT NOT NULL,
			PRIMARY KEY (groupId, userId)
		);

		CREATE TABLE IF NOT EXISTS reactions (
			messageId TEXT NOT NULL,
			reactorId TEXT NOT NULL,
			emoticon TEXT NOT NULL,
			PRIMARY KEY (messageId, reactorId, emoticon)
		);
		`

		_, err = db.Exec(schema)
		if err != nil {
			return nil, fmt.Errorf("error creating database schema: %w", err)
		}

	} else if err != nil {
		return nil, fmt.Errorf("error checking users table: %w", err)
	}

	return &appdbimpl{c: db}, nil
}
func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
