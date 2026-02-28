package auth

import (
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2"
)

// LoginHandler redirects to Google OAuth consent screen.
func LoginHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := "state-token" // in production use a random state and verify in callback
		url := cfg.OAuth2.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
		http.Redirect(w, r, url, http.StatusFound)
	}
}

// CallbackHandler exchanges the auth code for a token and stores it in the session, then redirects to callback URL.
func CallbackHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		tok, err := cfg.OAuth2.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "exchange failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		sess, err := cfg.Store.Get(r, SessionName)
		if err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		sess.Values[SessionTokenKey] = tok
		if err := sess.Save(r, w); err != nil {
			http.Error(w, "session save failed", http.StatusInternalServerError)
			return
		}
		redirect := cfg.Callback
		if redirect == "" {
			redirect = "/"
		}
		http.Redirect(w, r, redirect, http.StatusFound)
	}
}

// MeResponse is returned by MeHandler.
type MeResponse struct {
	Authenticated bool   `json:"authenticated"`
	Email         string `json:"email,omitempty"`
}

// MeHandler returns current user info if signed in.
func MeHandler(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := Token(r)
		if tok == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MeResponse{Authenticated: false})
			return
		}
		// We could decode the token's ID token for email; for simplicity just report authenticated.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MeResponse{Authenticated: true})
	}
}
