package cattp

import (
	"net/http"
)

// Handlers that re routes the request
// if Handlers are registered on the router, to them
// if they are not, to handler in the standard HTTP library
// ! Do not register other patterns other than generic ones like this
// ! Use "router.Handle"

func (router *Router[T]) handleNotFound(w http.ResponseWriter, r *http.Request, context T) {
	if router.notFound == nil {
		defer r.Body.Close()
		http.NotFound(w, r)
	} else {
		router.notFound.ServeHTTP(w, r, context)
	}
}

func (router *Router[T]) handleRoot(w http.ResponseWriter, r *http.Request, context T) {
	if router.root == nil {
		router.handleNotFound(w, r, context)
	} else {
		router.root.ServeHTTP(w, r, context)
	}
}
