package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"memegrab/cattp"
	"memegrab/sessions"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type profile struct {
	ID          int       `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Displayed   string    `json:"display_name"`
	IsOnline    bool      `json:"isOnline"`
	LastLogin   time.Time `json:"lastLogin"`
	LastOffline time.Time `json:"lastOffline"`
	IsAdmin     bool      `json:"isAdmin"`
}

type webapp struct {
	sessions sessions.SessionManager
	db       *sql.DB
}

type Payload struct {
	User    bool   `json:"user"`
	Message string `json:"message"`
}

// For URL use only domain name eg: google.it not https://google.it
func startWebApp(conf cattp.Config, db *sql.DB, sessions sessions.SessionManager) error {
	// httpAddr := fmt.Sprintf("%s:%s", conf.Host, conf.portPlain)
	context := &webapp{
		db:       db,
		sessions: sessions,
	}

	router := cattp.New(context)
	// router.HandleFunc("/", rootHandler)
	router.HandleFunc("/auth/signin", authHandler)
	router.HandleFunc("/auth", validateHandle)
	router.HandleFunc("/profile", profileHandle)
	router.HandleFunc("/saved", savedHandle)
	router.HandleFunc("/test", testHandler)

	err := router.Listen(&conf)
	if err != nil {
		panic("can't start webapp")
	}

	log.Println("HTTP Server succesfully started") // TODO: Move back in main func
	return nil
}

// var rootHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
// 	defer r.Body.Close()

// 	_, err := context.sessions.Validate(context.db, r)
// 	if err != nil {
// 		// TODO: Extend session upon device validation
// 		log.Println("Session error found")
// 		return
// 	}

// 	// This can be property slice of HTTP Instance
// 	index := filepath.Join("static", "app.html")
// 	temp := template.Must(template.New("app.html").ParseFiles(index))
// 	// _json, err := json.Marshal(profile)

// 	if err != nil {
// 		panic(err)
// 	}
// 	err = temp.Execute(w, nil)
// 	if err != nil {
// 		panic(err)
// 	}
// })

var authHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	defer r.Body.Close()

	dbSession, err := context.sessions.Validate(context.db, r)

	if err == nil {
		// TODO: Extend session upon device validation
		log.Println("Session found - redirecting to app")
		dbSession.SetClientCookie(w)
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var login *sessions.Credentials
	err = json.NewDecoder(r.Body).Decode(&login)
	if err != nil {
		panic(err)
	}

	loginDb, err := dbLogin(context.db, login.Email)
	if err != nil {
		log.Println("Can't get credentials from DB, wrong Email/Username")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(loginDb.Password), []byte(login.Password))
	if err != nil {
		log.Println("Incorrect password")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// TODO: Move to method in session manager
	jwtKey := []byte("palle")
	claims := &sessions.Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 720).Unix(),
		},
	}
	authToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := authToken.SignedString(jwtKey)
	session := context.sessions.Create(context.db, token, loginDb.ID, time.Time{})
	session.SetClientCookie(w) // TODO: Reimplement for JWT and WebSession
	if err != nil {
		log.Println("Error marshaling JWT Token")
		return
	}

	// TODO: Post response for WebSock?
	profile, err := userRead(context.db, session.UserId)
	if err != nil {
		log.Println("Can't find user profile")
	}
	payload, err := json.Marshal(profile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("%s succesfully logged in.", profile.Displayed)
	w.Write(payload)
})

var validateHandle = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	defer r.Body.Close()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	session, err := context.sessions.Validate(context.db, r)

	if err != nil {
		// TODO: Extend session upon device validation
		log.Println("Invalid session")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// session.SetClientCookie(w)
	// TODO: Post response for WebSock?
	w.Header().Add("Content-Type", "application/json")
	log.Printf("[%d][ID %v] Validated Session\n", http.StatusOK, session.UserId)
	w.WriteHeader(http.StatusOK)
})

var testHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte("Should be HTTP/2"))
})

var savedHandle = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	defer r.Body.Close()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	session, err := context.sessions.Validate(context.db, r)

	if err != nil {
		// TODO: Extend session upon device validation
		log.Println("Invalid session")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dbSaved := getDbMessages(context.db)
	saved, err := json.Marshal(dbSaved)
	if err != nil {
		panic(err)
	}
	log.Printf("[%d][ID %v] Get Saved Files\n", http.StatusOK, session.UserId)

	w.Header().Add("Content-Type", "application/json")
	w.Write(saved)
})

var profileHandle = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	defer r.Body.Close()

	if r.Method != http.MethodGet {
		log.Println("Invalid Method")
		return
	}

	session, err := context.sessions.Validate(context.db, r)

	if err != nil {
		// TODO: Extend session upon device validation
		log.Println("Unauthorized session")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// TODO: Post response for WebSock?
	profile, err := userRead(context.db, session.UserId)
	if err != nil {
		log.Println("Can't find user profile")
		w.WriteHeader(http.StatusBadRequest)
	}
	payload, err := json.Marshal(profile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Println("Found and sent profile")
	w.Write(payload)
})

// func notFound(w http.ResponseWriter, r *http.Request) {
// 	defer r.Body.Close()
// 	http.NotFound(w, r)
// }

// func favicon(w http.ResponseWriter, r *http.Request) {
// 	http.ServeFile(w, r, "static/images/favicon.ico")
// }

// func redirectToTls(w http.ResponseWriter, r *http.Request) {
// 	// log.Println("Redirected HTTP request to HTTPS")
// 	// http.Redirect(w, r, fmt.Sprintf("%s:%s", co)
// }
