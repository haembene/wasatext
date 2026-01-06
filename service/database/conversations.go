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

	out := []ConversationSnippet{}

	for rows.Next() {
		var id, name, photo, latestTs sql.NullString
		var isGroupInt sql.NullInt64

		if err := rows.Scan(&id, &name, &photo, &latestTs, &isGroupInt); err != nil {
			return nil, err
		}

		// Don't show empty chats
		if !latestTs.Valid {
			continue
		}

		isGroup := false
		if isGroupInt.Valid && isGroupInt.Int64 == 1 {
			isGroup = true
		}

		// get latest message
		var latestMsg Message
		if isGroup {
			latestMsg, err = db.getLatestMessageForGroup(id.String)
		} else {
			latestMsg, err = db.getLatestMessageForDirect(userID, id.String)
		}
		if err != nil {
			return nil, err
		}

		// count unread
		unread := 0
		if isGroup {
			// messages in a group, sent not by me and not read
			_ = db.c.QueryRow(`
                SELECT COUNT(*) FROM messages
                WHERE recipientId = ? AND senderId != ? AND status < 2
            `, id.String, userID).Scan(&unread)
		} else {
			// personal messages: i am receiver, they are sender, status < 2
			_ = db.c.QueryRow(`
                SELECT COUNT(*) FROM messages
                WHERE recipientId = ? AND senderId = ? AND status < 2
            `, userID, id.String).Scan(&unread)
		}

		sn := ConversationSnippet{
			ID:            id.String,
			Name:          name.String,
			PhotoUrl:      photo.String,
			LatestMessage: &latestMsg,
			IsGroup:       isGroup,
			UnreadCount:   unread,
		}

		out = append(out, sn)
	}

	// Check for errors after iteration
	if err := rows.Err(); err != nil {
		return nil, err
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
            SELECT id, senderId, timestamp, content, isPhoto, status, replyTo
            FROM messages
            WHERE recipientId = ?
            ORDER BY timestamp DESC
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
        SELECT id, senderId, timestamp, content, isPhoto, status, replyTo
        FROM messages
        WHERE (senderId = ? AND recipientId = ?)
           OR (senderId = ? AND recipientId = ?)
        ORDER BY timestamp DESC
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
        SELECT id, senderId, timestamp, content, isPhoto, status, replyTo
        FROM messages
        WHERE (senderId = ? AND recipientId = ?)
           OR (senderId = ? AND recipientId = ?)
        ORDER BY timestamp DESC
        LIMIT 1
    `, meID, otherID, otherID, meID)

	var m Message
	var replyTo sql.NullString
	if err := row.Scan(&m.ID, &m.SenderID, &m.Timestamp, &m.Content, &m.IsPhoto, &m.Status, &replyTo); err != nil {
		return Message{}, err
	}
	if replyTo.Valid {
		m.ReplyTo = replyTo.String
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
        SELECT id, senderId, timestamp, content, isPhoto, status, replyTo
        FROM messages
        WHERE recipientId = ?
        ORDER BY timestamp DESC
        LIMIT 1
    `, groupID)

	var m Message
	var replyTo sql.NullString
	if err := row.Scan(&m.ID, &m.SenderID, &m.Timestamp, &m.Content, &m.IsPhoto, &m.Status, &replyTo); err != nil {
		return Message{}, err
	}
	if replyTo.Valid {
		m.ReplyTo = replyTo.String
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

// reads rows (id, senderId, timestamp, content, isPhoto, status, replyTo)
// and adds reactions to them
func (db *appdbimpl) readMessagesWithReactions(rows *sql.Rows) ([]Message, error) {
	msgs := []Message{}
	for rows.Next() {
		var m Message
		var reply sql.NullString
		if err := rows.Scan(&m.ID, &m.SenderID, &m.Timestamp, &m.Content, &m.IsPhoto, &m.Status, &reply); err != nil {
			return nil, err
		}
		if reply.Valid {
			m.ReplyTo = reply.String
		}

		// load reactions
		rrows, err := db.c.Query(`SELECT reactorId, emoticon FROM reactions WHERE messageId = ?`, m.ID)
		if err != nil {
			return nil, err
		}

		// Iterate inner rows
		for rrows.Next() {
			var r Reaction
			if err := rrows.Scan(&r.ReactorID, &r.Emoticon); err != nil {
				_ = rrows.Close() // Fixed: Explicitly ignore close error
				return nil, err
			}
			m.Reactions = append(m.Reactions, r)
		}

		// Check for inner loop errors
		if err := rrows.Err(); err != nil {
			_ = rrows.Close() // Fixed: Explicitly ignore close error
			return nil, err
		}
		_ = rrows.Close() // Fixed: Explicitly ignore close error

		msgs = append(msgs, m)
	}

	// Check for outer loop errors
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return msgs, nil
}

// CreateConversation STUB
func (db *appdbimpl) CreateConversation(userID string, otherID string) (ConversationSnippet, error) {
	return ConversationSnippet{}, errors.New("CreateConversation not implemented")
}

// SetConversationRead STUB
func (db *appdbimpl) SetConversationRead(userID string, conversationID string) error {
	return errors.New("SetConversationRead not implemented")
}
