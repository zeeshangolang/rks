package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"time"

	"math/rand/v2"

	"golang.org/x/crypto/bcrypt"
)

type Tokens struct {
	Plaintext string
	Hash      string
	Expiry    time.Time
	Userid    int32
}

type TokenModel struct {
	DB *sql.DB
}

func NewRandomINt() (string, error) {

	rand := rand.IntN(999999)
	randInt := fmt.Sprintf("%06d", rand)
	return randInt, nil
}

func (m *TokenModel) InsertToken(userid int32, token string, expiry time.Time) error {

	hashed, err := bcrypt.GenerateFromPassword([]byte(token), 12)
	if err != nil {
		fmt.Println(err)
		return err
	}

	query := `INSERT INTO tokens (userid, hash, expiry) Values($1, $2, $3) `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = m.DB.ExecContext(ctx, query, userid, hashed, expiry)
	if err != nil {
		return err
	}
	return nil

}
func (m *TokenModel) UpdateTokenCount(userid int) (int, error) {

	var attempts int
	query := `UPDATE tokens SET attempts = attempts + 1 WHERE userid = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userid).Scan(&attempts)
	if err != nil {
		return 0, err
	}
	return attempts, nil
}

func (m *TokenModel) DeleteForAllUsers(userid int64) error {

	query := `DELETE * FROM tokens WHERE userid = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userid)
	return err

}

func (m *TokenModel) Confirmtoken(userid int, token string) (int, error) {

	var dbhash string

	query := `SELECT  hash FROM tokens JOIN users ON tokens.userid = users.id  WHERE userid = $1 
	AND tokens.expiry > NOW()`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userid).Scan(&dbhash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, ErrInvalidCredentials
		default:
			return 0, err
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(dbhash), []byte(token))
	if err != nil {
		return 0, ErrInvalidCredentials
	}
	return userid, nil
}
