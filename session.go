package main

import (
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type credentials struct {
	id       string
	username string
	password string
}

type session struct {
	userId  string
	token   string
	created time.Time
	expiry  time.Time
}

func (s *session) isExpired() bool {
	return s.expiry.Before(time.Now())
}

func validate(w http.ResponseWriter, r *http.Request, db *sql.DB) (*session, error) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil, err
	}

	token := cookie.Value

	session, err := read(db, token)
	if err != nil {
		log.Println("Cant get session")
		return nil, err
	}

	if session.isExpired() {
		delete(db, token)
		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   "",
			Expires: time.Now(),
		})
		log.Println("Session expired, removed from client")
	}
	return session, nil
}

func create(token string, id string) *session {
	session := &session{
		userId:  id,
		token:   token,
		created: time.Now(),
		expiry:  time.Now().Add(720 * time.Hour),
	}
	return session
}

// TODO: Move to bcrypt from b64
func saltedUUID(password string) string {
	saltSize := 32
	bRand := make([]byte, saltSize)
	_, err := rand.Read(bRand[:])
	if err != nil {
		log.Println("Can't read random bytes")
	}
	sha512Hasher := sha512.New()
	bPassword := []byte(password)
	bPassword = append(bPassword, bRand...)
	// TODO: Read again
	sha512Hasher.Write(bPassword)
	sum := sha512Hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(sum)
}

func matchHash(uuid string, password []byte, salt []byte) error {
	bRand := []byte(salt)
	bPassword := []byte(password)
	sha512Hasher := sha512.New()
	bPassword = append(bPassword, bRand...)
	// TODO: Read again
	sha512Hasher.Write(bPassword)
	sum := sha512Hasher.Sum(nil)
	encoded := base64.StdEncoding.EncodeToString(sum)
	if encoded != uuid {
		log.Println("Error matching salted password")
		return errors.New("err_salt_pass")
	}
	return nil
}

func read(db *sql.DB, token string) (*session, error) {
	sqlStatement := `SELECT * FROM http.sessions WHERE session_token=$1;`

	var id string
	var created time.Time
	var expires time.Time

	row := db.QueryRow(sqlStatement, token).Scan(&id, &token, &created, &expires)
	switch row {
	case sql.ErrNoRows:
		log.Println("No sessions found")
		return nil, sql.ErrNoRows

	case nil:
		session := &session{
			userId:  id,
			token:   token,
			created: created,
			expiry:  expires,
		}
		return session, nil
	default:
		log.Println("Error in reading sessions")
		return nil, errors.New("err_read_session")
	}
}

func delete(db *sql.DB, token string) error {
	sqlStatement := `DELTE FROM http.sessions WHERE session_token = $1;`
	res, err := db.Exec(sqlStatement, token)
	if err != nil {
		log.Println("Error in deleting session")
	}
	_, err = res.RowsAffected()
	if err != nil {
		log.Println("No sessions deleted")
	}
	log.Printf("Deleted token %s", token)
	return nil
}

// TODO: MUST be a db function
func store(db *sql.DB, id string, token string, created time.Time, expires time.Time) (err error) {
	sqlStatement := `
	INSERT INTO public.http_sessions (user_id, expires, session_token, created)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (user_id) DO UPDATE
		SET user_id = excluded.user_id,
			expires = excluded.expires,
			session_token = excluded.session_token,
			created = excluded.created;`
	db.QueryRow(sqlStatement, id, expires, token, created)
	log.Println("Saved new session")
	return
}
