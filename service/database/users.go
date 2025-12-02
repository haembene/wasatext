package database

import (
	"database/sql"
	"strings"

	"github.com/google/uuid"
)

type User struct {
	ID       string
	Name     string
	PhotoUrl string
}

func (db *appdbimpl) LoginOrCreateUser(name string) (string, error) {
	var existingID string

	err := db.c.QueryRow(
		"SELECT id FROM users WHERE name = ?",
		name,
	).Scan(&existingID)

	if err == nil {
		return existingID, nil // user exists
	}

	if err != sql.ErrNoRows {
		return "", err // real DB error
	}

	// Create new user
	newID := strings.ReplaceAll(uuid.New().String(), "-", "")
	defaultPhoto := "" // or some default image URL

	_, err = db.c.Exec(
		"INSERT INTO users (id, name, photoUrl) VALUES (?, ?, ?)",
		newID, name, defaultPhoto,
	)
	if err != nil {
		return "", err
	}

	return newID, nil
}

func (db *appdbimpl) FindUsers(prefix string) ([]User, error) {
	rows, err := db.c.Query(
		"SELECT id, name, photoUrl FROM users WHERE name LIKE ? LIMIT 50",
		prefix+"%",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []User

	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.Name, &u.PhotoUrl)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}

	return result, nil
}

func (db *appdbimpl) IsUserNameTaken(name string) (bool, error) {
	var id string
	err := db.c.QueryRow(
		"SELECT id FROM users WHERE name = ?",
		name,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return false, nil // name is free
	}

	if err != nil {
		return false, err // real DB error
	}

	return true, nil
}

func (db *appdbimpl) UpdateUserName(userID, newName string) error {
	_, err := db.c.Exec(
		"UPDATE users SET name = ? WHERE id = ?",
		newName, userID,
	)
	return err
}

func (db *appdbimpl) UpdateUserPhoto(userID, url string) error {
	_, err := db.c.Exec(
		"UPDATE users SET photoUrl = ? WHERE id = ?",
		url, userID,
	)
	return err
}
