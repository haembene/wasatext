package database

import (
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"strings"
)

// ErrForbidden is used when a user is not authorized to perform an action.
// Defining it here resolves the "repeated string literal" linter error.
var ErrForbidden = errors.New("forbidden")

// CHECK MEMBERSHIP
// GetGroupMember checks if a user is a member of the specified group.
// Used for access validation.
func (db *appdbimpl) GetGroupMember(groupID, userID string) (bool, error) {
	var tmp string
	// Query: Select userId from group_members matching group ID and user ID
	err := db.c.QueryRow(`
        SELECT userId FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, userID).Scan(&tmp)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil // Not a member
	}
	if err != nil {
		return false, err // DB error
	}
	return true, nil // User is a member
}

// AddUserToGroup adds a new user to an existing group.
// Validates: 1) Group exists, 2) Target user exists, 3) Requester is currently a member of the group.
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

	// add: insert membership record into group_members
	_, err = db.c.Exec(`
        INSERT INTO group_members (groupId, userId)
        VALUES (?, ?)
    `, groupID, userIDToAdd)

	if err != nil {
		return err
	}

	return nil
}

// LeaveGroup removes a member from a group.
// Validates: 1) Requester matches target ID (only can leave yourself), 2) Group exists, 3) User is currently in the group.
func (db *appdbimpl) LeaveGroup(groupID, userID, requesterID string) error {

	// check if user we want to delete is THIS user (cannot eject others, only leave group ourselves)
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

	// delete member: remove row from group_members
	_, err = db.c.Exec(`
        DELETE FROM group_members
        WHERE groupId = ? AND userId = ?
    `, groupID, userID)

	if err != nil {
		return err
	}

	return nil
}

// SetGroupName renames a group.
// Validates: 1) Group exists, 2) Requester is a group member, 3) Group name does not conflict with another group.
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

	// check if there exists another group with the same name to prevent naming conflicts
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

	// group name update: update groups metadata table
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

// SetGroupPhoto uploads a base64 profile photo for the group.
// Validates: 1) Group exists, 2) Requester is a member of the group.
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

	// photo update: store base64 string in database groups table
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

// CreateGroup creates a new group conversation and adds the creator as a participant.
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

// GetGroupMembers retrieves a list of all users in a group.
// Validates: Requester must be a member of the group to see other members.
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

	// Query: SQL Join to retrieve member names and photos from users table matching group ID
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
