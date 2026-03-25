package database

import (
	"database/sql"
	"errors"
	
	"strings"
	"github.com/google/uuid"
)

// ErrForbidden is used when a user is not authorized to perform an action.
// Defining it here resolves the "repeated string literal" linter error.
var ErrForbidden = errors.New("forbidden")

// CHECK MEMBERSHIP
func (db *appdbimpl) GetGroupMember(groupID, userID string) (bool, error) {
	var tmp string
	err := db.c.QueryRow(`
        SELECT userId FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, userID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *appdbimpl) AddUserToGroup(groupID, userIDToAdd, requesterID string) error {

	// check if group exists
	var tmp string
	err := db.c.QueryRow(`
        SELECT id FROM groups WHERE id = ?
    `, groupID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("group not found")
	}
	if err != nil {
		return err
	}

	// check if user exists
	err = db.c.QueryRow(`
        SELECT id FROM users WHERE id = ?
    `, userIDToAdd).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("user not found")
	}
	if err != nil {
		return err
	}

	// check if the adder is group member
	isMember, err := db.GetGroupMember(groupID, requesterID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrForbidden
	}

	// add
	_, err = db.c.Exec(`
        INSERT INTO group_members (groupId, userId)
        VALUES (?, ?)
    `, groupID, userIDToAdd)

	if err != nil {
		return err
	}

	return nil
}

func (db *appdbimpl) LeaveGroup(groupID, userID, requesterID string) error {

	// check if user we want to delete is THIS user
	if userID != requesterID {
		return ErrForbidden
	}

	var tmp string
	err := db.c.QueryRow(`
        SELECT id FROM groups WHERE id = ?
    `, groupID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("group not found")
	}
	if err != nil {
		return err
	}

	// check if user is group member
	err = db.c.QueryRow(`
        SELECT userId 
        FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, userID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("user not in group")
	}
	if err != nil {
		return err
	}

	// delete member
	_, err = db.c.Exec(`
        DELETE FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, userID)

	if err != nil {
		return err
	}

	return nil
}

func (db *appdbimpl) SetGroupName(groupID, requesterID, newName string) error {

	// check if group exists
	var tmp string
	err := db.c.QueryRow(`
        SELECT id FROM groups WHERE id = ?
    `, groupID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("group not found")
	}
	if err != nil {
		return err
	}

	// requester - group member?
	err = db.c.QueryRow(`
        SELECT userId
        FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, requesterID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}

	// check if there exists another group with the same name
	err = db.c.QueryRow(`
        SELECT id FROM groups 
        WHERE name = ? AND id != ?
    `, newName, groupID).Scan(&tmp)

	if err == nil {
		// another group with the same name - conflict
		return errors.New("conflict") // API → 409
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// group name update
	_, err = db.c.Exec(`
        UPDATE groups
        SET name = ?
        WHERE id = ?
    `, newName, groupID)

	if err != nil {
		return err
	}

	return nil
}

func (db *appdbimpl) SetGroupPhoto(groupID, requesterID, newPhotoURL string) error {

	var tmp string
	err := db.c.QueryRow(`
        SELECT id FROM groups WHERE id = ?
    `, groupID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("group not found")
	}
	if err != nil {
		return err
	}

	// requesterID — group member?
	err = db.c.QueryRow(`
        SELECT userId
        FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, requesterID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}

	// photo update
	_, err = db.c.Exec(`
        UPDATE groups
        SET photoUrl = ?
        WHERE id = ?
    `, newPhotoURL, groupID)

	if err != nil {
		return err
	}

	return nil
}

// CreateGroup creates a new group conversation and adds the creator as a participant
func (db *appdbimpl) CreateGroup(name string, ownerID string) (Conversation, error) {
	var group Conversation
	groupID := strings.ReplaceAll(uuid.New().String(), "-", "")

	// 1. Insert into groups table
	_, err := db.c.Exec(`INSERT INTO groups (id, name, photoUrl) VALUES (?, ?, '')`, groupID, name)
	if err != nil {
		return group, err
	}

	// 2. Add the creator to group_members table
	_, err = db.c.Exec(`INSERT INTO group_members (groupId, userId) VALUES (?, ?)`, groupID, ownerID)
	if err != nil {
		return group, err
	}

	// 3. Return the new group object
	group.ID = groupID
	group.Name = name
	group.IsGroup = true
	group.PhotoUrl = ""

	return group, nil
}

func (db *appdbimpl) GetGroupMembers(groupID string, userID string) ([]User, error) {
	// check if user is a member
	var tmp string
	err := db.c.QueryRow(`
		SELECT userId FROM group_members 
		WHERE groupId = ? AND userId = ?
	`, groupID, userID).Scan(&tmp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("forbidden")
		}
		return nil, err
	}

	rows, err := db.c.Query(`
		SELECT u.id, u.name, u.photoUrl 
		FROM group_members gm
		JOIN users u ON gm.userId = u.id
		WHERE gm.groupId = ?
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var members []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.PhotoURL); err != nil {
			return nil, err
		}
		members = append(members, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}
