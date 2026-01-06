package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PhotoURL string `json:"photoUrl"`
}

type Reaction struct {
	ReactorID string `json:"reactorId"`
	Emoticon  string `json:"emoticon"`
}

// Conversation represents a chat or group conversation
type Conversation struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PhotoUrl string `json:"photoUrl"`
	IsGroup  bool   `json:"isGroup"`
}

// Message represents a single message
type Message struct {
	ID        string     `json:"id"`
	SenderID  string     `json:"senderId"`
	Content   string     `json:"content"`
	Timestamp string     `json:"timestamp"`
	IsPhoto   bool       `json:"isPhoto"`
	Status    int        `json:"status"`
	ReplyTo   string     `json:"replyTo,omitempty"`
	Reactions []Reaction `json:"reactions"`
}

// ConversationSnippet is used for the list of conversations
type ConversationSnippet struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	PhotoUrl      string   `json:"photoUrl"`
	LatestMessage *Message `json:"latestMessage"`
	IsGroup       bool     `json:"isGroup"`
	UnreadCount   int      `json:"unreadCount"`
}

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
	// NOTE: We changed ownerID to string to match the rest of your app
	CreateGroup(name string, ownerID string) (Conversation, error)
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

		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			is_group BOOLEAN NOT NULL,
			photo_url TEXT NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS participants (
			user_id TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			PRIMARY KEY (user_id, conversation_id)
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
