package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id          int64
	Name        string
	Email       string
	PassworHash string
	Activated   bool
	Version     int32
	ProfileImg  sql.NullString
}

var AnonymusUser = &User{}

func (u *User) IsAnonymus() bool {
	return u == AnonymusUser
}

func (u *User) IsActived() bool {
	return u.Activated
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string) (int, error) {
	var id int
	hashedpass, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, err
	}
	query := `INSERT INTO users(name, email, password_hash, activated) 
	VALUES($1, $2, $3, false) RETURNING id`

	err = m.DB.QueryRow(query, name, email, hashedpass).Scan(&id)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return 0, ErrDuplicateemail
		default:
			return 0, err
		}

	}
	return id, nil
}

func (m *UserModel) Authenticate(email, password string) (int, bool, error) {
	var IsActivated bool
	var hashedPass []byte
	var id int
	query := `SELECT id, password_hash, activated FROM users WHERE 	email = $1`

	err := m.DB.QueryRow(query, email).Scan(&id, &hashedPass, &IsActivated)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, false, ErrInvalidCredentials
		default:
			return 0, false, err
		}
	}
	err = bcrypt.CompareHashAndPassword(hashedPass, []byte(password))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return 0, false, ErrInvalidCredentials
		default:
			return 0, false, err
		}
	}

	return id, IsActivated, nil

}

func (m *UserModel) Exists(id int) (error, bool) {
	var exists bool

	query := `SELECT EXISTS(SELECT true FROM users WHERE id = $1)`
	err := m.DB.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return err, false
	}
	return nil, exists

}

func (m *UserModel) Latest() ([]*User, error) {
	query := ` SELECT name , email FROM  users ORDER BY id LIMIT 10`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		s := &User{}
		err := rows.Scan(&s.Name, &s.Email)
		if err != nil {
			return nil, err
		}
		users = append(users, s)
	}
	return users, nil
}

func (m *UserModel) UpdateUser(id int) (*User, error) {

	query := `UPDATE users SET activated = true Where id = $1 RETURNING 
	name, activated`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	s := &User{}
	err := m.DB.QueryRowContext(ctx, query, id).Scan(&s.Name, &s.Activated)
	if err != nil {
		return nil, err
	}

	fmt.Print(s)
	return s, nil
}

func (m *UserModel) GetUSERByName(name string) ([]*User, error) {
	query := `SELECT name , id  FROM users WHERE name = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	users := []*User{}

	rows, err := m.DB.QueryContext(ctx, query, name)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrecordNotFound
		default:
			return nil, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		s := &User{}

		err := rows.Scan(&s.Name, &s.Id)
		if err != nil {
			return nil, err
		}
		users = append(users, s)
	}

	return users, nil
}
