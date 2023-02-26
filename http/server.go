package http

import (
	"database/sql"
	"log"
	"memegrab/sessions"
	"net/http"
	"path/filepath"
	"text/template"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Config struct {
	Host       string
	portPlain  string
	portSecure string
	URL        string
	crtPath    string
	keyPath    string
}

func Start(conf *Config, router *Router) *http.Server {
	h2s := &http2.Server{}
	h1s := &http.Server{
		Addr:    conf.Host + conf.portPlain,
		Handler: h2c.NewHandler(router, h2s),
	}
	log.Panic(h1s.ListenAndServe())
	return h1s
}

// For URL use only domain name eg: google.it not https://google.it

func startHTTPServer(conf Config, db *sql.DB) error {
	// httpAddr := fmt.Sprintf("%s:%s", conf.Host, conf.portPlain)

	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		login(w, r, db)
	})

	log.Fatal(h1s.ListenAndServe())

	log.Println("HTTP Server succesfully started") // TODO: Move back in main func
	return nil
}

func index(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// This can be property slice of HTTP Instance
	index := filepath.Join("static", "app.html")
	temp := template.Must(template.New("index.html").ParseFiles(index))

	err := temp.Execute(w, nil)
	if err != nil {
		panic(err)
	}
}

func login(w http.ResponseWriter, r *http.Request, db *sql.DB) error {
	defer r.Body.Close()

	_, err := Validate(w, r, db)
	if err == nil {
		// TODO: Extend session upon device validation
		log.Println("Session found - redirecting to app")
		http.Redirect(w, r, "/", http.StatusFound)
		return err
	}

	if r.Method == http.MethodPost {
		login := sessions.Credentials{
			Username: r.PostForm.Get("username"),
			Password: r.PostForm.Get("password"),
		}

		loginDb, err := dbLogin(db, login.username)
		if err != nil {
			log.Println("Can't get credentials from DB")
		}
		err = bcrypt.CompareHashAndPassword([]byte(loginDb.password), []byte(login.password))
		if err != nil {
			log.Println("Incorrect password")
			return err
		}
		token := saltedUUID(login.password)
		session := create(token, loginDb.id)

		// TODO: Define initial profile setup?
		// profile, err := getDbUser(db, loginDb.id)
		// if err != nil {
		// 	log.Println("Can't find user profile")
		// }

		err = store(db, session.userId, session.token, session.created, session.expiry)
		if err != nil {
			log.Println("Can't store session")
		}

		http.SetCookie(w, &http.Cookie{
			Name:    "session_token",
			Value:   session.token,
			Expires: session.expiry,
		})
		// TODO: Post response for WebSock?
		http.Redirect(w, r, "/", http.StatusFound)
		return err
	}
	// TODO: Templates (If even to be used) must be generated elsewhere prior and reused (http custom type property?)
	if r.Method == http.MethodGet {
		loginPage := filepath.Join("static", "login.html")
		template := template.Must(template.New("login.html").ParseFiles(loginPage))

		err = template.Execute(w, nil)
		if err != nil {
			log.Println("Error excuting template")
			return err
		}
	}
	// TODO: Better include returning error
	return nil
}

// func test(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Add("Content-Type", "text/html")
// 	w.Write([]byte("Should be HTTP/2"))
// }

func notFound(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	http.NotFound(w, r)
}

func favicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/images/favicon.ico")
}

// func redirectToTls(w http.ResponseWriter, r *http.Request) {
// 	// log.Println("Redirected HTTP request to HTTPS")
// 	// http.Redirect(w, r, fmt.Sprintf("%s:%s", co)
// }
