package main

import (
	"database/sql"
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
	router.HandleFunc("/", root[*webapp])
	router.HandleFunc("/login", loginHandler)
	router.HandleFunc("/test", testHandler)

	err := router.Listen(&conf)
	if err != nil {
		panic("can't start webapp")
	}

	log.Println("HTTP Server succesfully started") // TODO: Move back in main func
	return nil
}

func root[T any](w http.ResponseWriter, r *http.Request, context T) {
	defer r.Body.Close()
	// This can be property slice of HTTP Instance
	index := filepath.Join("static", "app.html")
	temp := template.Must(template.New("app.html").ParseFiles(index))

	err := temp.Execute(w, nil)
	if err != nil {
		panic(err)
	}
}

var loginHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	defer r.Body.Close()

	session, err := context.sessions.Validate(context.db, r)
	if err == nil {
		// TODO: Extend session upon device validation
		log.Println("Session found - redirecting to app")
		session.SetClientCookie(w)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	// // TODO: Define initial profile setup?
	// profile, err := userRead(context.db, session.UserId)
	// if err != nil {
	// 	log.Println("Can't find user profile")
	// }
	// // Assigning
	// profile.ID = session.UserId

	if r.Method == http.MethodPost {
		login := sessions.Credentials{
			Username: r.PostForm.Get("username"),
			Password: r.PostForm.Get("password"),
		}

		loginDb, err := dbLogin(context.db, login.Username)
		if err != nil {
			log.Println("Can't get credentials from DB")
		}
		err = bcrypt.CompareHashAndPassword([]byte(loginDb.Password), []byte(login.Password))
		if err != nil {
			log.Println("Incorrect password")
			return
		}
		token := sessions.SaltedUUID(login.Password) // TODO: Should this be a method of SessionManager?
		session := context.sessions.Create(context.db, token, session.UserId, time.Time{})

		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   session.Token,
			Expires: session.Expiry,
		})
		// // TODO: Post response for WebSock?
		// http.Redirect(w, r, "/", http.StatusFound)
		// return err
	}
	// TODO: Templates (If even to be used) must be generated elsewhere prior and reused (http custom type property?)
	if r.Method == http.MethodGet {
		loginPage := filepath.Join("static", "login.html")
		template := template.Must(template.New("login.html").ParseFiles(loginPage))

		err := template.Execute(w, nil)
		if err != nil {
			log.Println("Error excuting template")
		}
	}
})

var testHandler = cattp.HandlerFunc[*webapp](func(w http.ResponseWriter, r *http.Request, context *webapp) {
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte("Should be HTTP/2"))
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
