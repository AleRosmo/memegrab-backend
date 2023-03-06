package cattp

import (
	"fmt"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Defining custom interface with ServeHTTP signature containing custom parameter
// to register custom handlers with DB
// We need to expect not anymore "http.Handler" but simply Handler
type Handler[T any] interface {
	ServeHTTP(http.ResponseWriter, *http.Request, T)
}

// Used to type any function that has this signature into a type
// that implemets the Handler interface, to allow it to be registerd on the mux
// with a corresponding pattern. Upon pattern match, this function will be invoked it
// call the user's defined callback function 'f' with all the arguments passed from
// the multiplexer adapter middleware (the one that turns the standard http.Handler signature
// with our custom ServeHTTP with db argument)
type HandlerFunc[T any] func(http.ResponseWriter, *http.Request, T)

func (f HandlerFunc[T]) ServeHTTP(w http.ResponseWriter, r *http.Request, c T) {
	f(w, r, c)
}

type Config struct {
	Host string
	Port string
	URL  string
	// portSecure string
	// crtPath    string
	// keyPath    string
}

func New[T any](context T) *Router[T] {
	router := &Router[T]{
		Mux:      http.NewServeMux(),
		Context:  context,
		root:     nil, // If we assign this they will be used by the handler functions instead of the usual http.Handler
		notFound: nil,
		// DB:       db,
	}
	//TODO: Dynamically register subfolders
	//TODO: eg. ask what is dir to be served
	//TODO:		-> ls directory
	//TODO:     -> get subdirectories
	//TODO:		-> register subdir handler like below with "http.Dir" using
	//TODO:     ..filepath.join with "provided_dir" + "found dir name"  )
	router.Mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("static/css"))))
	router.Mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("static/js"))))

	router.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			router.handleRoot(w, r, context)
		} else {
			router.handleNotFound(w, r, context)
		}
	})
	return router
}

type Router[T any] struct {
	Mux     *http.ServeMux
	Context T
	// DB       *sql.DB
	// Sessions sessions.SessionManager
	root     Handler[T]
	notFound Handler[T]
}

func (router *Router[T]) Handle(pattern string, handler Handler[T]) {
	if handler == nil {
		panic("Empty handler")
	}

	if pattern == "/" {
		if router.root != nil {
			panic("Root pattern already registered")
		}
		router.root = handler
	} else {
		router.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			handler.ServeHTTP(w, r, router.Context)
		})
	}
}

func (router *Router[T]) HandleFunc(pattern string, handler func(w http.ResponseWriter, r *http.Request, context T)) {
	if handler == nil {
		panic("Empty handler")
	}
	// TODO: add check for existing handler
	router.Handle(pattern, HandlerFunc[T](handler))

}

func (router *Router[T]) Listen(conf *Config) error {
	h2s := &http2.Server{}
	h1s := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", conf.Host, conf.Port),
		Handler: h2c.NewHandler(router, h2s),
	}
	err := (h1s.ListenAndServe())
	return err
}

// Allows the Router to behave as Handler for incoming HTTP Requests by
// wrapping the it's internal Mux Handler, acting as middleware.
func (router *Router[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.Mux.ServeHTTP(w, r)
}

// Register on our Mux the Handler Type provided in args for "Not Found"
// It must be a Handler Type, otherwise use 'NotFoundHandleFunc'
func (router *Router[T]) NotFoundHandle(handler Handler[T]) {
	if handler == nil {
		panic("Empty handler")
	}

	if router.notFound == nil {
		router.notFound = handler
	} else {
		panic("Not Found pattern already registered")
	}
}

// Commodity wrapper for registering a custom passed Not Found function
// calls the 'NotFoundHandle' function, but allow us to pass a custom function
// that matches the signature.
func (router *Router[T]) NotFoundHandleFunc(handler func(http.ResponseWriter, *http.Request, T)) {
	if handler == nil {
		panic("Empty handler")
	}
	// Type our function as our custom HandlerFunc, it implements
	// the custom Handler interface having the ServeHTTP method with
	// custom signature
	router.NotFoundHandle(HandlerFunc[T](handler))
}
