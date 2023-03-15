package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"memegrab/cattp"
	"memegrab/sessions"
	"net/http"
	"path/filepath"
	"text/template"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type profile struct {
	ID          int       `json:"user_id"`
	Username    string    `json:"username"`
	Permissions []string  `json:"isAdmin"`
	DisplayName string    `json:"display_name"`
	IsOnline    bool      `json:"isOnline"`
	LastLogin   time.Time `json:"lastLogin"`
	LastOffline time.Time `json:"lastOffline"`
}

type webapp struct {
	sessions sessions.SessionManager
	db       *sql.DB
}

// For URL use only domain name eg: google.it not https://google.it
func startWebApp(conf cattp.Config, db *sql.DB, sessions sessions.SessionManager) error {
	// httpAddr := fmt.Sprintf("%s:%s", conf.Host, conf.portPlain)
	context := &webapp{
		db:       db,
		sessions: sessions,
	}

	router := cattp.New(context)
	router.HandleFunc("/", rootHandler)
	router.HandleFunc("/auth", authHandler)
	router.HandleFunc("/saved", getSavedHandler)
	router.HandleFunc("/test", testHandler)

	err := router.Listen(&conf)
	if err != nil {
		panic("can't start webapp")
	}

	log.Println("HTTP Server succesfully started") // TODO: Move back in main func
	return nil
}

var rootHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	defer r.Body.Close()

	_, err := context.sessions.Validate(context.db, r)
	if err != nil {
		// TODO: Extend session upon device validation
		log.Println("Session error found - redirecting to login")
		http.Redirect(w, r, "http://localhost:3000/login", http.StatusFound)
		return
	}
	// profile, err := userRead(context.db, session.UserId)
	// if err != nil {
	// 	panic(err)
	// }

	// This can be property slice of HTTP Instance
	index := filepath.Join("static", "app.html")
	temp := template.Must(template.New("app.html").ParseFiles(index))
	// _json, err := json.Marshal(profile)

	if err != nil {
		panic(err)
	}
	err = temp.Execute(w, nil)
	if err != nil {
		panic(err)
	}
})

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

	// profile, err := userRead(context.db, session.UserId)
	// if err != nil {
	// 	log.Println("Can't find user profile")
	// }
	// // Assigning
	// profile.ID = session.UserId

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
	token := sessions.SaltedUUID(login.Password) // TODO: Should this be a method of SessionManager?
	session := context.sessions.Create(context.db, token, loginDb.ID, time.Time{})
	session.SetClientCookie(w)
	// TODO: Post response for WebSock?
	w.Write([]byte("k bro"))
})

var testHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte("Should be HTTP/2"))
})

// var authHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
// 	r.ParseForm()
// 	data := r.PostForm
// 	//!!!!!! REMOVEEEE !!!!
// 	w.Header().Set("Access-Control-Allow-Origin", "*")
// 	w.Write([]byte(fmt.Sprintf("Received: %v", data)))
// })

var getSavedHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {

	if r.Method != http.MethodGet {
		return
	}
	defer r.Body.Close()

	dbSaved := getDbMessages(context.db)

	saved, err := json.Marshal(dbSaved)
	if err != nil {
		panic(err)
	}
	w.Write(saved)
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
