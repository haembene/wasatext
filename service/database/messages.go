package database

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (db *appdbimpl) SendMessage(senderID, recipientID, content string, replyTo *string) (Message, error) {

	if !db.recipientExists(recipientID) {
		return Message{}, errors.New("recipient not found")
	}

	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	timestamp := time.Now().UTC().Format(time.RFC3339)

	isPhoto := isPhotoContent(content)

	// save message
	// Note: We pass the 'replyTo' pointer directly to Exec.
	// If it is nil, SQL inserts NULL. If it has a value, SQL inserts the string.
	_, err := db.c.Exec(`
		INSERT INTO messages (id, senderId, recipientId, timestamp, content, isPhoto, status, replyTo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`,
		id, senderID, recipientID, timestamp, content, isPhoto, 0, replyTo,
	)
	if err != nil {
		return Message{}, err
	}

	// Prepare string value for the struct
	var replyToVal string
	if replyTo != nil {
		replyToVal = *replyTo
	}

	// return message
	msg := Message{
		ID:        id,
		SenderID:  senderID,
		Timestamp: timestamp,
		Content:   content,
		IsPhoto:   isPhoto,
		Status:    0,
		ReplyTo:   replyToVal, // <--- FIXED: Assigns string (not *string)
		Reactions: []Reaction{},
	}

	return msg, nil
}

func (db *appdbimpl) ForwardMessage(messageID, newRecipientID, senderID string) (Message, error) {

	// load original message
	var original Message
	var replyTo sql.NullString

	err := db.c.QueryRow(`
		SELECT id, senderId, timestamp, content, isPhoto, status, replyTo
		FROM messages
		WHERE id = ?
		LIMIT 1;
	`, messageID).Scan(
		&original.ID,
		&original.SenderID,
		&original.Timestamp,
		&original.Content,
		&original.IsPhoto,
		&original.Status,
		&replyTo,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Message{}, errors.New("message not found")
	}
	if err != nil {
		return Message{}, err
	}

	// FIXED: Assign string value to struct
	if replyTo.Valid {
		original.ReplyTo = replyTo.String
	} else {
		original.ReplyTo = ""
	}

	// check if receipient exists
	if !db.recipientExists(newRecipientID) {
		return Message{}, errors.New("recipient not found")
	}

	// create new message
	newID := strings.ReplaceAll(uuid.New().String(), "-", "")
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Determine what to pass to SQL for ReplyTo
	var newReplyToPtr *string
	if original.ReplyTo != "" {
		newReplyToPtr = &original.ReplyTo
	}

	_, err = db.c.Exec(`
		INSERT INTO messages (id, senderId, recipientId, timestamp, content, isPhoto, status, replyTo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`,
		newID,
		senderID, // person who forwards
		newRecipientID,
		timestamp,
		original.Content, // copy text
		original.IsPhoto, // copy type
		0,                // sent
		newReplyToPtr,    // <--- FIXED: Pass pointer or nil to SQL
	)

	if err != nil {
		return Message{}, err
	}

	// return new message
	forwarded := Message{
		ID:        newID,
		SenderID:  senderID,
		Timestamp: timestamp,
		Content:   original.Content,
		IsPhoto:   original.IsPhoto,
		Status:    0,
		ReplyTo:   original.ReplyTo, // This is already a string
		Reactions: []Reaction{},
	}

	return forwarded, nil
}

func (db *appdbimpl) DeleteMessage(messageID, userID string) error {

	// Check if the message exists and belongs to userID
	var sender string
	err := db.c.QueryRow(`
		SELECT senderId 
		FROM messages 
		WHERE id = ?
		LIMIT 1
	`, messageID).Scan(&sender)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("message not found")
	}
	if err != nil {
		return err
	}

	if sender != userID {
		return errors.New("forbidden") // FIXED: Removed undefined ErrForbidden
	}

	// delete reactions
	_, err = db.c.Exec(`
		DELETE FROM reactions 
		WHERE messageId = ?
	`, messageID)
	if err != nil {
		return err
	}

	// delete message
	_, err = db.c.Exec(`
		DELETE FROM messages 
		WHERE id = ?
	`, messageID)
	if err != nil {
		return err
	}

	return nil
}

func (db *appdbimpl) AddReaction(messageID, userID, emoticon string) error {

	// check if message exists
	var exists string
	err := db.c.QueryRow(`
		SELECT id FROM messages WHERE id = ?
	`, messageID).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("message not found")
	}
	if err != nil {
		return err
	}

	// make reaction
	_, err = db.c.Exec(`
		INSERT OR IGNORE INTO reactions (messageId, reactorId, emoticon)
		VALUES (?, ?, ?)
	`, messageID, userID, emoticon)

	if err != nil {
		return err
	}

	return nil
}

func (db *appdbimpl) RemoveReaction(messageID, userID, emoticon string) error {

	var exists string
	err := db.c.QueryRow(`
		SELECT id FROM messages WHERE id = ?
	`, messageID).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("message not found")
	}
	if err != nil {
		return err
	}

	// delete reaction
	res, err := db.c.Exec(`
		DELETE FROM reactions
		WHERE messageId = ? AND reactorId = ? AND emoticon = ?
	`, messageID, userID, emoticon)

	if err != nil {
		return err
	}

	// check if there was a reaction
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("reaction not found")
	}

	return nil
}

func (db *appdbimpl) recipientExists(id string) bool {
	var tmp string

	// check user
	err := db.c.QueryRow(`SELECT id FROM users WHERE id = ?`, id).Scan(&tmp)
	if err == nil {
		return true
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false
	}

	// check group
	err = db.c.QueryRow(`SELECT id FROM groups WHERE id = ?`, id).Scan(&tmp)
	return err == nil
}

func isPhotoContent(content string) bool {
	lower := strings.ToLower(content)

	if strings.HasPrefix(lower, "http") &&
		(strings.HasSuffix(lower, ".jpg") ||
			strings.HasSuffix(lower, ".jpeg") ||
			strings.HasSuffix(lower, ".png") ||
			strings.HasSuffix(lower, ".gif") ||
			strings.HasSuffix(lower, ".webp")) {
		return true
	}

	return false
}
