package main

import (
	"bytes"
	"calendar/server"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

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
	reqPath := r.URL.Path
	if reqPath == "/" {
		reqPath = "/index.html"
	}
	// embed.FS.Open needs path without leading slash
	cleanPath := strings.TrimPrefix(reqPath, "/")
	if cleanPath == "" {
		cleanPath = "index.html"
	}
	f, err := s.static.Open(cleanPath)
	if err != nil {
		// SPA fallback: serve index.html for unknown paths
		serveIndex(w, r, s.static)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		serveIndex(w, r, s.static)
		return
	}
	// Serve file directly to avoid FileServer's /index.html -> / redirect loop
	serveFile(w, r, f, stat, cleanPath)
}

func serveIndex(w http.ResponseWriter, r *http.Request, fsys http.FileSystem) {
	f, err := fsys.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	stat, _ := f.Stat()
	serveFile(w, r, f, stat, "index.html")
}

func serveFile(w http.ResponseWriter, r *http.Request, f io.Reader, stat fs.FileInfo, name string) {
	ext := path.Ext(name)
	ct := "application/octet-stream"
	switch ext {
	case ".html":
		ct = "text/html; charset=utf-8"
	case ".js":
		ct = "application/javascript"
	case ".css":
		ct = "text/css"
	}
	w.Header().Set("Content-Type", ct)
	content, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	modTime := stat.ModTime()
	if modTime.IsZero() {
		modTime = time.Now()
	}
	http.ServeContent(w, r, stat.Name(), modTime, bytes.NewReader(content))
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

	staticRoot, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal().Err(err).Msg("unable to get static root")
	}
	spaHandler := spaFileServer{static: http.FS(staticRoot)}
	r.PathPrefix("/").Handler(spaHandler)

	log.Info().Msg("starting server on port 8123")

	err = http.ListenAndServe(":8123", r)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}

}
