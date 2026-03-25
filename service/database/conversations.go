package database

import (
	"database/sql"
	"errors"
)

func (db *appdbimpl) GetMyConversations(userID string, usernameFilter *string) ([]ConversationSnippet, error) {
	var rows *sql.Rows
	var err error

	// Collect UNION: users (direct) + groups, then sort by latestTs DESC
	query := `
            SELECT id, name, photoUrl, latestTs, isGroup FROM (
              SELECT 
                u.id AS id,
                u.name AS name,
                u.photoUrl AS photoUrl,
                (
                  SELECT timestamp
                  FROM messages m2
                  WHERE (m2.senderId = ? AND m2.recipientId = u.id)
                     OR (m2.senderId = u.id AND m2.recipientId = ?)
                  ORDER BY timestamp DESC
                  LIMIT 1
                ) AS latestTs,
                0 AS isGroup
              FROM users u
              WHERE u.id IN (
                  SELECT DISTINCT CASE WHEN senderId = ? THEN recipientId ELSE senderId END
                  FROM messages
                  WHERE senderId = ? OR recipientId = ?
              )
              UNION ALL
              SELECT 
                g.id AS id,
                g.name AS name,
                g.photoUrl AS photoUrl,
                (
                  SELECT timestamp
                  FROM messages m2
                  WHERE m2.recipientId = g.id
                  ORDER BY timestamp DESC
                  LIMIT 1
                ) AS latestTs,
                1 AS isGroup
              FROM groups g
              JOIN group_members gm ON gm.groupId = g.id
              WHERE gm.userId = ?
            )
            ORDER BY latestTs DESC`

	if usernameFilter == nil {
		rows, err = db.c.Query(query, userID, userID, userID, userID, userID, userID)
	} else {
		filterQuery := `
            SELECT id, name, photoUrl, latestTs, isGroup FROM (
              SELECT 
                u.id AS id,
                u.name AS name,
                u.photoUrl AS photoUrl,
                (
                  SELECT timestamp
                  FROM messages m2
                  WHERE (m2.senderId = ? AND m2.recipientId = u.id)
                     OR (m2.senderId = u.id AND m2.recipientId = ?)
                  ORDER BY timestamp DESC
                  LIMIT 1
                ) AS latestTs,
                0 AS isGroup
              FROM users u
              WHERE u.name LIKE ? AND u.id != ?
              UNION ALL
              SELECT 
                g.id AS id,
                g.name AS name,
                g.photoUrl AS photoUrl,
                (
                  SELECT timestamp
                  FROM messages m2
                  WHERE m2.recipientId = g.id
                  ORDER BY timestamp DESC
                  LIMIT 1
                ) AS latestTs,
                1 AS isGroup
              FROM groups g
              JOIN group_members gm ON gm.groupId = g.id
              WHERE gm.userId = ?
            )
            ORDER BY latestTs DESC`
		rows, err = db.c.Query(filterQuery, userID, userID, (*usernameFilter)+"%", userID, userID)
	}

	if err != nil {
		return nil, err
	}
	// Fixed: Explicitly ignore error in defer to satisfy errcheck
	defer func() { _ = rows.Close() }()

	// 5. Read all basic info first to free the connection
	type convRow struct {
		id       string
		name     string
		photo    string
		isGroup  bool
	}
	var cRows []convRow

	for rows.Next() {
		var id, name, photo, latestTs sql.NullString
		var isGroupInt sql.NullInt64

		if err := rows.Scan(&id, &name, &photo, &latestTs, &isGroupInt); err != nil {
			return nil, err
		}

		isGroup := false
		if isGroupInt.Valid && isGroupInt.Int64 == 1 {
			isGroup = true
		}

		cRows = append(cRows, convRow{
			id:      id.String,
			name:    name.String,
			photo:   photo.String,
			isGroup: isGroup,
		})
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Close rows explicitly before making nested queries
	rows.Close()

	out := []ConversationSnippet{}

	// 6. Do nested queries without holding the outer Rows lock
	for _, cr := range cRows {
		// get latest message
		var latestMsg Message
		var pointerMsg *Message
		if cr.isGroup {
			latestMsg, err = db.getLatestMessageForGroup(cr.id)
		} else {
			latestMsg, err = db.getLatestMessageForDirect(userID, cr.id)
		}

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				pointerMsg = nil
			} else {
				return nil, err
			}
		} else {
			pointerMsg = &latestMsg
		}

		// count unread
		unread := 0
		if cr.isGroup {
			// messages in a group, sent not by me and not read
			_ = db.c.QueryRow(`
                SELECT COUNT(*) FROM messages
                WHERE recipientId = ? AND senderId != ? AND status < 2
            `, cr.id, userID).Scan(&unread)
		} else {
			// personal messages: i am receiver, they are sender, status < 2
			_ = db.c.QueryRow(`
                SELECT COUNT(*) FROM messages
                WHERE recipientId = ? AND senderId = ? AND status < 2
            `, userID, cr.id).Scan(&unread)
		}

		sn := ConversationSnippet{
			ID:            cr.id,
			Name:          cr.name,
			PhotoUrl:      cr.photo,
			LatestMessage: pointerMsg,
			IsGroup:       cr.isGroup,
			UnreadCount:   unread,
		}

		out = append(out, sn)
	}

	return out, nil
}

// GetConversation returns the history of messages between me and conversationID.
func (db *appdbimpl) GetConversation(meID, conversationID string) ([]Message, error) {

	// Check if the user with specified id exists
	var tmp string
	err := db.c.QueryRow(`SELECT id FROM users WHERE id = ?`, conversationID).Scan(&tmp)
	isUser := (err == nil)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if !isUser {
		// group
		err = db.c.QueryRow(`SELECT id FROM groups WHERE id = ?`, conversationID).Scan(&tmp)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("conversation not found")
			}
			return nil, err
		}
		rows, err := db.c.Query(`
            SELECT m.id, m.senderId, m.timestamp, m.content, m.isPhoto, m.status, m.replyTo, u.name, u.photoUrl
            FROM messages m
            LEFT JOIN users u ON m.senderId = u.id
            WHERE m.recipientId = ?
            ORDER BY m.timestamp DESC
            LIMIT 100
        `, conversationID)
		if err != nil {
			return nil, err
		}
		// Fixed: Explicitly ignore error in defer
		defer func() { _ = rows.Close() }()

		return db.readMessagesWithReactions(rows)
	}

	// direct chat
	rows, err := db.c.Query(`
        SELECT m.id, m.senderId, m.timestamp, m.content, m.isPhoto, m.status, m.replyTo, u.name, u.photoUrl
        FROM messages m
        LEFT JOIN users u ON m.senderId = u.id
        WHERE (m.senderId = ? AND m.recipientId = ?)
           OR (m.senderId = ? AND m.recipientId = ?)
        ORDER BY m.timestamp DESC
        LIMIT 100
    `, meID, conversationID, conversationID, meID)
	if err != nil {
		return nil, err
	}
	// Fixed: Explicitly ignore error in defer
	defer func() { _ = rows.Close() }()

	return db.readMessagesWithReactions(rows)
}

func (db *appdbimpl) getLatestMessageForDirect(meID, otherID string) (Message, error) {
	row := db.c.QueryRow(`
        SELECT m.id, m.senderId, m.timestamp, m.content, m.isPhoto, m.status, m.replyTo, u.name, u.photoUrl
        FROM messages m
        LEFT JOIN users u ON m.senderId = u.id
        WHERE (m.senderId = ? AND m.recipientId = ?)
           OR (m.senderId = ? AND m.recipientId = ?)
        ORDER BY m.timestamp DESC
        LIMIT 1
    `, meID, otherID, otherID, meID)

	var m Message
	var replyTo, senderName, senderPhoto sql.NullString
	if err := row.Scan(&m.ID, &m.SenderID, &m.Timestamp, &m.Content, &m.IsPhoto, &m.Status, &replyTo, &senderName, &senderPhoto); err != nil {
		return Message{}, err
	}
	if replyTo.Valid {
		m.ReplyTo = replyTo.String
	}
	if senderName.Valid {
		m.SenderName = senderName.String
	}
	if senderPhoto.Valid {
		m.SenderPhoto = senderPhoto.String
	}

	// load reactions
	rrows, err := db.c.Query(`SELECT reactorId, emoticon FROM reactions WHERE messageId = ?`, m.ID)
	if err != nil {
		return Message{}, err
	}
	// Fixed: Explicitly ignore error in defer
	defer func() { _ = rrows.Close() }()

	for rrows.Next() {
		var r Reaction
		if err := rrows.Scan(&r.ReactorID, &r.Emoticon); err != nil {
			return Message{}, err
		}
		m.Reactions = append(m.Reactions, r)
	}

	// Check for errors after iteration
	if err := rrows.Err(); err != nil {
		return Message{}, err
	}

	return m, nil
}

func (db *appdbimpl) getLatestMessageForGroup(groupID string) (Message, error) {
	row := db.c.QueryRow(`
        SELECT m.id, m.senderId, m.timestamp, m.content, m.isPhoto, m.status, m.replyTo, u.name, u.photoUrl
        FROM messages m
        LEFT JOIN users u ON m.senderId = u.id
        WHERE m.recipientId = ?
        ORDER BY m.timestamp DESC
        LIMIT 1
    `, groupID)

	var m Message
	var replyTo, senderName, senderPhoto sql.NullString
	if err := row.Scan(&m.ID, &m.SenderID, &m.Timestamp, &m.Content, &m.IsPhoto, &m.Status, &replyTo, &senderName, &senderPhoto); err != nil {
		return Message{}, err
	}
	if replyTo.Valid {
		m.ReplyTo = replyTo.String
	}
	if senderName.Valid {
		m.SenderName = senderName.String
	}
	if senderPhoto.Valid {
		m.SenderPhoto = senderPhoto.String
	}

	// load reactions
	rrows, err := db.c.Query(`SELECT reactorId, emoticon FROM reactions WHERE messageId = ?`, m.ID)
	if err != nil {
		return Message{}, err
	}
	// Fixed: Explicitly ignore error in defer
	defer func() { _ = rrows.Close() }()

	for rrows.Next() {
		var r Reaction
		if err := rrows.Scan(&r.ReactorID, &r.Emoticon); err != nil {
			return Message{}, err
		}
		m.Reactions = append(m.Reactions, r)
	}

	// Check for errors after iteration
	if err := rrows.Err(); err != nil {
		return Message{}, err
	}

	return m, nil
}

func (db *appdbimpl) readMessagesWithReactions(rows *sql.Rows) ([]Message, error) {
	// 1. Read all messages into memory first to avoid nested database cursors lock (Deadlock prevention)
	msgs := []Message{}
	for rows.Next() {
		var m Message
		var reply, senderName, senderPhoto sql.NullString
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Timestamp, &m.Content, &m.IsPhoto, &m.Status, &reply, &senderName, &senderPhoto); err != nil {
			return nil, err
		}
		if reply.Valid {
			m.ReplyTo = reply.String
		}
		if senderName.Valid {
			m.SenderName = senderName.String
		}
		if senderPhoto.Valid {
			m.SenderPhoto = senderPhoto.String
		}
		msgs = append(msgs, m)
	}

	// Check for outer loop errors
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// We must close rows before issuing nested queries to avoid connection pool starvation / locking
	rows.Close()

	// 2. Fetch reactions for the loaded messages
	for i := range msgs {
		m := &msgs[i]

		// load reactions
		rrows, err := db.c.Query(`
			SELECT r.reactorId, r.emoticon, u.name 
			FROM reactions r
			LEFT JOIN users u ON r.reactorId = u.id
			WHERE r.messageId = ?
		`, m.ID)
		if err != nil {
			return nil, err
		}

		// Iterate inner rows
		for rrows.Next() {
			var r Reaction
			var reactorName sql.NullString
			if err := rrows.Scan(&r.ReactorID, &r.Emoticon, &reactorName); err != nil {
				_ = rrows.Close() 
				return nil, err
			}
			if reactorName.Valid {
				r.ReactorName = reactorName.String
			}
			m.Reactions = append(m.Reactions, r)
		}

		// Check for inner loop errors
		if err := rrows.Err(); err != nil {
			_ = rrows.Close() 
			return nil, err
		}
		_ = rrows.Close() 
	}

	return msgs, nil
}

// CreateConversation returns the target user snippet for a direct chat
func (db *appdbimpl) CreateConversation(userID string, otherID string) (ConversationSnippet, error) {
	var name, photoUrl string
	err := db.c.QueryRow(`SELECT name, photoUrl FROM users WHERE id = ?`, otherID).Scan(&name, &photoUrl)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ConversationSnippet{}, errors.New("user not found")
		}
		return ConversationSnippet{}, err
	}

	return ConversationSnippet{
		ID:          otherID,
		Name:        name,
		PhotoUrl:    photoUrl,
		IsGroup:     false,
		UnreadCount: 0,
	}, nil
}

// SetConversationRead marks conversations as read (status = 2)
func (db *appdbimpl) SetConversationRead(userID string, conversationID string) error {
	var tmp string
	// Check if the conversation is a group
	err := db.c.QueryRow(`SELECT id FROM groups WHERE id = ?`, conversationID).Scan(&tmp)
	if err == nil {
		// 1. Mark as read by this user
		_, err = db.c.Exec(`
			INSERT OR IGNORE INTO message_reads (messageId, userId)
			SELECT id, ? FROM messages 
			WHERE recipientId = ? AND senderId != ? AND status < 2
		`, userID, conversationID, userID)
		if err != nil {
			return err
		}

		// 2. Upgrade to status=2 for messages where all other members have read it
		_, err = db.c.Exec(`
			UPDATE messages
			SET status = 2
			WHERE recipientId = ? AND status < 2 AND (
				SELECT COUNT(*) FROM message_reads WHERE messageId = messages.id
			) >= (SELECT COUNT(*) FROM group_members WHERE groupId = ?) - 1
		`, conversationID, conversationID)
		return err
	}

	// Direct message
	_, err = db.c.Exec(`
		UPDATE messages
		SET status = 2
		WHERE recipientId = ? AND senderId = ? AND status < 2
	`, userID, conversationID)
	return err
}
