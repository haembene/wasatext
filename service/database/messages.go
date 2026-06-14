package database

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SendMessage creates a message record in the messages table.
// Inputs: senderID (UUID), recipientID (UUID of user or group), content (string), replyTo (optional pointer to original message UUID).
// DB Action: Runs INSERT INTO messages. It defaults status to 0 (Sent).
func (db *appdbimpl) SendMessage(senderID, recipientID, content string, replyTo *string) (Message, error) {

	if !db.recipientExists(recipientID) {
		return Message{}, errors.New("recipient not found")
	}

	// Verify group membership of sender if recipient is a group
	var isGroup bool
	var tmp string
	err := db.c.QueryRow(`SELECT id FROM groups WHERE id = ?`, recipientID).Scan(&tmp)
	if err == nil {
		isGroup = true
	}
	if isGroup {
		var memberCount int
		err = db.c.QueryRow(`
			SELECT COUNT(*) FROM group_members 
			WHERE groupId = ? AND userId = ?
		`, recipientID, senderID).Scan(&memberCount)
		if err != nil {
			return Message{}, err
		}
		if memberCount == 0 {
			return Message{}, errors.New("forbidden")
		}
	}

	id := strings.ReplaceAll(uuid.New().String(), "-", "")
	timestamp := time.Now().UTC().Format(time.RFC3339)

	isPhoto := isPhotoContent(content)

	// save message: Insert query with status 0
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

	// Prepare SenderName struct definition
	var senderName string
	_ = db.c.QueryRow(`SELECT name FROM users WHERE id = ?`, senderID).Scan(&senderName)

	// return message
	msg := Message{
		ID:         id,
		SenderID:   senderID,
		SenderName: senderName,
		Timestamp:  timestamp,
		Content:    content,
		IsPhoto:    isPhoto,
		Status:     0,
		ReplyTo:    replyToVal, // <--- FIXED: Assigns string (not *string)
		Reactions:  []Reaction{},
	}

	return msg, nil
}

// ForwardMessage copies an existing message and forwards it to a recipient.
// Validates: 1) Original message exists, 2) Target exists, 3) Preexisting chat history exists between sender and target.
// DB Action: Retrieves original content, verifies constraints, prepends |FORWARDED| to the text, and inserts a copy.
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

	// Validate preexisting chat constraint
	var isGroup bool
	var tmp string
	err = db.c.QueryRow(`SELECT id FROM groups WHERE id = ?`, newRecipientID).Scan(&tmp)
	if err == nil {
		isGroup = true
	}

	if isGroup {
		var memberCount int
		err = db.c.QueryRow(`
			SELECT COUNT(*) FROM group_members 
			WHERE groupId = ? AND userId = ?
		`, newRecipientID, senderID).Scan(&memberCount)
		if err != nil {
			return Message{}, err
		}
		if memberCount == 0 {
			return Message{}, errors.New("no preexisting chat with this recipient")
		}
	} else {
		var msgCount int
		err = db.c.QueryRow(`
			SELECT COUNT(*) FROM messages 
			WHERE (senderId = ? AND recipientId = ?) 
			   OR (senderId = ? AND recipientId = ?)
		`, senderID, newRecipientID, newRecipientID, senderID).Scan(&msgCount)
		if err != nil {
			return Message{}, err
		}
		if msgCount == 0 {
			return Message{}, errors.New("no preexisting chat with this recipient")
		}
	}

	// create new message
	newID := strings.ReplaceAll(uuid.New().String(), "-", "")
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Determine what to pass to SQL for ReplyTo
	var newReplyToPtr *string
	if original.ReplyTo != "" {
		newReplyToPtr = &original.ReplyTo
	}

	// Inject forwarding markup string
	fwdContent := "|FORWARDED|" + original.Content

	_, err = db.c.Exec(`
		INSERT INTO messages (id, senderId, recipientId, timestamp, content, isPhoto, status, replyTo)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
	`,
		newID,
		senderID, // person who forwards
		newRecipientID,
		timestamp,
		fwdContent,       // copy text WITH markers
		original.IsPhoto, // copy type
		0,                // sent
		newReplyToPtr,    // Pass pointer or nil to SQL
	)

	if err != nil {
		return Message{}, err
	}

	var senderName string
	_ = db.c.QueryRow(`SELECT name FROM users WHERE id = ?`, senderID).Scan(&senderName)

	// return new message
	forwarded := Message{
		ID:         newID,
		SenderID:   senderID,
		SenderName: senderName,
		Timestamp:  timestamp,
		Content:    fwdContent, // Display active
		IsPhoto:    original.IsPhoto,
		Status:     0,
		ReplyTo:    original.ReplyTo,
		Reactions:  []Reaction{},
	}

	return forwarded, nil
}

// DeleteMessage deletes a message from the database.
// Validates: 1) Message exists, 2) Requester is the sender of the message.
// DB Action: Removes associated reactions from the reactions table, then deletes the message row.
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

	// delete reactions: Clear children rows first
	_, err = db.c.Exec(`
		DELETE FROM reactions 
		WHERE messageId = ?
	`, messageID)
	if err != nil {
		return err
	}

	// delete message: Clear actual row
	_, err = db.c.Exec(`
		DELETE FROM messages 
		WHERE id = ?
	`, messageID)
	if err != nil {
		return err
	}

	return nil
}

// AddReaction attaches an emoji to a message.
// Validates: 1) Message exists, 2) User can have at most ONE reaction per message.
// DB Action: Deletes any pre-existing reaction by this user on the message, then inserts the new reaction emoji.
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

	// enforce constraint: user can add NO MORE THAN ONE reaction per message
	_, err = db.c.Exec(`
		DELETE FROM reactions WHERE messageId = ? AND reactorId = ?
	`, messageID, userID)
	if err != nil {
		return err
	}

	// make reaction: Insert query
	_, err = db.c.Exec(`
		INSERT INTO reactions (messageId, reactorId, emoticon)
		VALUES (?, ?, ?)
	`, messageID, userID, emoticon)

	if err != nil {
		return err
	}

	return nil
}

// RemoveReaction detaches a user's emoji from a message.
// Validates: 1) Message exists, 2) Target reaction exists.
// DB Action: Runs DELETE query on the reactions table.
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

	// delete reaction: delete row matching composite key values
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

// recipientExists validates if a recipient ID represents a valid user or group in the database.
func (db *appdbimpl) recipientExists(id string) bool {
	var tmp string

	// check user
	err := db.c.QueryRow(`SELECT id FROM users WHERE id = ?`, id).Scan(&tmp)
	if err == nil {
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false
	}

	// check group
	err = db.c.QueryRow(`SELECT id FROM groups WHERE id = ?`, id).Scan(&tmp)
	return err == nil
}

// isPhotoContent inspects content to determine if it should be marked as a media photo.
func isPhotoContent(content string) bool {
	lower := strings.ToLower(content)

	if strings.Contains(content, "|PHOTO|data:image") {
		return true
	}

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
