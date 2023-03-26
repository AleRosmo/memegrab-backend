package sessions

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
)

type Credentials struct {
	ID       int    `json:"id"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Claims struct {
	jwt.StandardClaims
}

type Auth struct {
	UserId  int
	Token   string
	Created time.Time
	Expiry  time.Time
}

type Token = string
type UserID = int
type SessionLenght = time.Time

type SessionManager interface {
	Create(*sql.DB, Token, UserID, SessionLenght) *session
	Delete(*sql.DB, string) error
	Validate(*sql.DB, *http.Request) (*session, error)
	Read(*sql.DB, string) (*session, error)
}

func New(dl time.Duration) *Manager {
	return &Manager{
		defaultLenght:  dl,
		activeSessions: make([]*session, 0),
	}
}

type Manager struct {
	defaultLenght  time.Duration
	activeSessions []*session
}

// TODO: Some decent in error checking would be nice
func (sm *Manager) Create(db *sql.DB, token Token, id UserID, lenght SessionLenght) *session {
	// TODO: lenght to become Time.Duration and evaluate isZero
	var _lenght time.Time = time.Now().Add(sm.defaultLenght)
	if !lenght.IsZero() {
		_lenght = lenght
	}
	session := &session{
		UserId:  id,
		Token:   token,
		Created: time.Now(),
		Expiry:  _lenght,
	}

	// TODO: MUST be a db function
	sqlStatement := `
	INSERT INTO http.sessions (id, expiry, token, created)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (id) DO UPDATE
		SET id = excluded.id,
			expiry = excluded.expiry,
			token = excluded.token,
			created = excluded.created;
	`
	row := db.QueryRow(sqlStatement, id, _lenght, token, session.Created)
	if err := row.Err(); err != nil {
		return nil
	}
	sm.activeSessions = append(sm.activeSessions, session)
	log.Println("Saved new session")
	return session
}

// If returns error 'nil' valid ?
// TODO: Add user id to cookies 'somehow'
func (sm *Manager) Validate(db *sql.DB, r *http.Request) (*session, error) {

	cookie, err := r.Cookie("memegrab")
	if err != nil {
		return nil, err
	}
	err = cookie.Valid()
	if err != nil {
		return nil, err
	}

	// TODO: Further cookie properties check
	var userSession *session
	var idx int
	for i, session := range sm.activeSessions {
		if session.Token == cookie.Value {
			userSession = session
			idx = i
			// Continues to search session in DB
			log.Println("Found stored active session")
		}
	}

	// TODO: Search on DB if not found
	// ? If loading from DB data, this is 'redundant'
	// Server might have restarted, search DB and compare
	if userSession == nil {
		userSession, err = sm.Read(db, cookie.Value)
		if err != nil {
			log.Println("Error getting session from DB")
			// return nil, err
		}
		if userSession == nil {
			log.Println("Session not found on DB")
			return nil, err
		}
	}

	if userSession.isExpired() {
		sm.Delete(db, userSession.Token)

		// Remove from activeSessions slice
		sm.activeSessions[idx] = sm.activeSessions[len(sm.activeSessions)-1]
		sm.activeSessions[len(sm.activeSessions)-1] = nil
		sm.activeSessions = sm.activeSessions[:len(sm.activeSessions)-1]

		log.Println("Session expired, removed")
		return nil, errors.New("expired")
	}

	// newLenght := time.Now().Add(sm.defaultLenght)
	// sm.activeSessions[idx].Expiry = newLenght

	// // TODO: MUST be a db function
	// query := `
	// 	UPDATE http.sessions
	// 		SET expiry = $2
	// 	WHERE id = $1;
	// `
	// _, err = db.Exec(query, userSession.UserId, newLenght)
	// if err != nil {
	// 	log.Println("Error updating session lenght")
	// 	return nil, errors.New("err_upd_session")
	// }
	return userSession, nil
}

func (sm *Manager) Read(db *sql.DB, token Token) (*session, error) {
	sqlStatement := `SELECT * FROM http.sessions WHERE token=$1;`

	var id int
	var created time.Time
	var expires time.Time

	row := db.QueryRow(sqlStatement, token).Scan(&id, &token, &created, &expires)
	switch row {
	case sql.ErrNoRows:
		log.Println("No sessions found")
		return nil, sql.ErrNoRows

	case nil:
		session := &session{
			UserId:  id,
			Token:   token,
			Created: created,
			Expiry:  expires,
		}
		return session, nil
	default:
		log.Println("Error in reading sessions")
		return nil, errors.New("err_read_session")
	}
}

func (sm *Manager) Delete(db *sql.DB, token Token) error {
	sqlStatement := `DELETE FROM http.sessions WHERE token = $1;`
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

type session struct {
	UserId  int
	Token   string
	Created time.Time
	Expiry  time.Time
}

func (s *session) SetClientCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "memegrab",
		Value:   s.Token,
		Expires: s.Expiry,
		// SameSite: http.SameSiteNoneMode,
		// Secure:   true, //! SET AGAIN WHEN USING HTTPS
		HttpOnly: true,
		Path:     "/",
	})
}

func (s *session) isExpired() bool {
	return s.Expiry.Before(time.Now())
}

// func store(db *sql.DB, id string, token string, created time.Time, expires time.Time) (err error) {

// 	return
// }
