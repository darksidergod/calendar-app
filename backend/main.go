package main

import (
	"calendar/server"
	"embed"
	"io/fs"

	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2/google"
	googlecalendarv3 "google.golang.org/api/calendar/v3"
)

//go:embed static/*
var staticFS embed.FS

type spaFileServer struct {
	static http.FileSystem
}

func (s spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	f, err := s.static.Open(path)
	if err != nil {
		r2 := *r
		r2.URL = &url.URL{Path: "/index.html"}
		http.FileServer(s.static).ServeHTTP(w, &r2)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		r2 := *r
		r2.URL = &url.URL{Path: "/index.html"}
		http.FileServer(s.static).ServeHTTP(w, &r2)
		return
	}
	r2 := *r
	r2.URL = &url.URL{Path: path}
	http.FileServer(s.static).ServeHTTP(w, &r2)
}

func main() {
	// TODO; refactor token to be fetched from the webapp
	// TODO; refactor credentials to be stored in a secret manager

	// refactor -start- //
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatal().Err(err).Msg("unable to read client secret file")
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, googlecalendarv3.CalendarScope)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to parse client secret file to config")
	}

	server := server.NewCalendarServer(config)

	r := mux.NewRouter()
	r.HandleFunc("/listevents", server.ListEventsHandler)
	r.HandleFunc("/createevents", server.CreateEventsHandler)
	r.HandleFunc("/deleteevent", server.DeleteEventHandler)
	r.HandleFunc("/callback", server.AuthCallbackHandler)
	r.HandleFunc("/authurl", server.GetAuthURLHandler)

	staticRoot, _ := fs.Sub(staticFS, "static")
	spaHandler := spaFileServer{static: http.FS(staticRoot)}
	r.PathPrefix("/").Handler(spaHandler)

	log.Info().Msg("starting server on port 8123")

	err = http.ListenAndServe(":8123", r)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

}
