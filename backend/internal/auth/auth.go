package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	google_calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	SessionName     = "calendar-app-session"
	SessionTokenKey = "oauth2_token"
	SessionUserKey  = "user_email"
)

var (
	// OAuth2 scopes; CalendarScope allows full calendar access.
	Scopes = []string{google_calendar.CalendarScope, "email", "profile"}
)

// Config holds OAuth and session configuration.
type Config struct {
	OAuth2   *oauth2.Config
	Store    sessions.Store
	Callback string // redirect URL after login, e.g. "http://localhost:8080"
}

// Token returns the stored oauth2.Token for the request, or nil if not authenticated.
func Token(r *http.Request) *oauth2.Token {
	sess, err := GetSession(r)
	if err != nil {
		return nil
	}
	v := sess.Values[SessionTokenKey]
	if v == nil {
		return nil
	}
	tok, _ := v.(*oauth2.Token)
	return tok
}

var defaultStore sessions.Store
var defaultStoreOnce sync.Once

// GetSession returns the session for the request (store from context or default).
func GetSession(r *http.Request) (*sessions.Session, error) {
	store := r.Context().Value(contextKeyStore{})
	if store != nil {
		return store.(sessions.Store).Get(r, SessionName)
	}
	defaultStoreOnce.Do(func() {
		authKey, encKey := sessionKeys()
		defaultStore = sessions.NewCookieStore(authKey, encKey)
	})
	return defaultStore.Get(r, SessionName)
}

// SetDefaultStore sets the package-level session store (call from NewConfig so Token/GetSession work without middleware).
func SetDefaultStore(store sessions.Store) {
	defaultStore = store
}

type contextKeyStore struct{}

// Middleware injects the session store into context and optionally requires auth.
func Middleware(cfg *Config, requireAuth bool) func(http.Handler) http.Handler {
	store := cfg.Store
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKeyStore{}, store)
			r = r.WithContext(ctx)
			if requireAuth {
				sess, _ := store.Get(r, SessionName)
				tokVal := sess.Values[SessionTokenKey]
				if tokVal == nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"not authenticated"}`))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NewConfig loads OAuth config from credentials.json and builds session store from env.
func NewConfig(redirectURL, callbackURL string) (*Config, error) {
	credPath := os.Getenv("GOOGLE_CREDENTIALS_PATH")
	if credPath == "" {
		credPath = "credentials.json"
	}
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, err
	}
	oauth2Config, err := google.ConfigFromJSON(b, Scopes...)
	if err != nil {
		return nil, err
	}
	oauth2Config.RedirectURL = redirectURL
	authKey, encKey := sessionKeys()
	store := sessions.NewCookieStore(authKey, encKey)
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteLaxMode
	if callbackURL != "" && len(callbackURL) >= 5 && callbackURL[:5] == "https" {
		store.Options.Secure = true
	}
	SetDefaultStore(store)
	return &Config{OAuth2: oauth2Config, Store: store, Callback: callbackURL}, nil
}

func sessionKeys() ([]byte, []byte) {
	authB64 := os.Getenv("SESSION_AUTH_KEY")
	encB64 := os.Getenv("SESSION_ENCRYPT_KEY")
	if authB64 != "" && encB64 != "" {
		auth, _ := base64.StdEncoding.DecodeString(authB64)
		enc, _ := base64.StdEncoding.DecodeString(encB64)
		if len(auth) >= 32 && len(enc) >= 16 {
			return auth[:32], enc[:16]
		}
	}
	// Fallback: single secret hashed (for dev only; prefer explicit keys in prod)
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production"
	}
	auth := make([]byte, 32)
	enc := make([]byte, 16)
	for i := 0; i < 32; i++ {
		auth[i] = secret[i%len(secret)]
	}
	for i := 0; i < 16; i++ {
		enc[i] = secret[(i+7)%len(secret)]
	}
	return auth, enc
}

// CalendarClient returns a Google Calendar API client for the given token.
func CalendarClient(ctx context.Context, token *oauth2.Token, cfg *Config) (*google_calendar.Service, error) {
	client := cfg.OAuth2.Client(ctx, token)
	return google_calendar.NewService(ctx, option.WithHTTPClient(client))
}
