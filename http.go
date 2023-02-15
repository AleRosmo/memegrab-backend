package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"text/template"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// For URL use only domain name eg: google.it not https://google.it
type httpConf struct {
	host       string
	portPlain  string
	portSecure string
	URL        string
	crtPath    string
	keyPath    string
}

func startHTTPServer(conf httpConf) error {
	httpAddr := fmt.Sprintf("%s:%s", conf.host, conf.portPlain)
	mux := http.NewServeMux()

	mux.HandleFunc("/test", test)
	mux.HandleFunc("/index", index)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			notFound(w, r)
		} else {
			index(w, r)
		}
	})

	handler := http.HandlerFunc(mux.ServeHTTP)

	h2s := &http2.Server{}
	h1s := &http.Server{
		Addr:    httpAddr,
		Handler: h2c.NewHandler(handler, h2s),
	}

	log.Fatal(h1s.ListenAndServe())

	// httpServer
	// log.Println("Starting HTTP Server ")

	// fs := http.FileServer(http.Dir("./static"))
	// http.Handle("/static/", http.StripPrefix("/static/", fs))

	// http.HandleFunc("/favicon.ico", favicon)
	// http.HandleFunc("/", root)

	// httpsAddr := fmt.Sprintf("%s:%s", conf.host, conf.portSecure)

	// httpUrl := fmt.Sprintf("http://%s", conf.URL)
	// httpsUrl := fmt.Sprintf("https://%s", conf.URL)

	// HTTP Listener
	// wg.Add(1)
	// go func() {
	// 	err := http.ListenAndServe(httpAddr, nil)
	// 	if err != nil {
	// 		log.Println("Failed to start HTTP listener")
	// 		return
	// 	}
	// 	log.Println("Started HTTP listener")
	// }()

	// // HTTPS Listener
	// wg.Add(1)
	// go func() {
	// 	log.Println("Failed to start HTTPS listener")
	// 	err := http.ListenAndServeTLS(httpsAddr, conf.crtPath, conf.keyPath, nil)
	// 	if err != nil {
	// 		return
	// 	}
	// 	log.Println("Started HTTPS listener")
	// }()
	log.Println("HTTP Server succesfully started") // TODO: Move back in main func
	return nil
}

func index(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// This can be property slice of HTTP Instance
	index := filepath.Join("static", "index.html")
	temp := template.Must(template.New("index.html").ParseFiles(index))

	err := temp.Execute(w, nil)
	if err != nil {
		panic(err)
	}
}

func test(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte("Should be HTTP/2"))
}

func notFound(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	http.NotFound(w, r)
}

// func favicon(w http.ResponseWriter, r *http.Request) {
// 	http.ServeFile(w, r, "static/images/favicon.ico")
// }

// func redirectToTls(w http.ResponseWriter, r *http.Request) {
// 	// log.Println("Redirected HTTP request to HTTPS")
// 	// http.Redirect(w, r, fmt.Sprintf("%s:%s", co)
// }
