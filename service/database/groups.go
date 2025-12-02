package database

import (
	"database/sql"
	"errors"
)

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
		return errors.New("forbidden")
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
		return errors.New("forbidden")
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
		return errors.New("forbidden")
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
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
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
		return errors.New("forbidden")
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
