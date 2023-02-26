package http

import (
	"database/sql"
	"net/http"
)

type Router struct {
	mux      *http.ServeMux
	db       *sql.DB
	root     http.Handler
	notFound http.Handler
}

func New(db *sql.DB) (*Router, error) {
	router := &Router{
		mux:      http.NewServeMux(),
		root:     nil,
		notFound: nil,
		// sessions: sessions.New(time.Now().Add(time.Hour * 720)),
	}
	router.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			router.rootHandler(w, r)
		} else {
			router.notFoundHandler(w, r)
		}
	})
	// r := http.HandleFu

	return router, nil

}

func (router *Router) notFoundHandler(w http.ResponseWriter, r *http.Request) {

}

func (router *Router) rootHandler(w http.ResponseWriter, r *http.Request) {
	if router.root == nil {

	}
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.mux.ServeHTTP(w, r)
}
