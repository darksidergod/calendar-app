package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
)

var (
	tokenFile   = "token.json"
	redirectURL = "http://localhost:8123/callback"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) (*http.Client, error) {
	token, err := tokenFromFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("no authentication token found")
	}

	return config.Client(context.Background(), token), nil
}

func getTokenURL(config *oauth2.Config) string {
	config.RedirectURL = redirectURL
	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func saveTokenFromWeb(config *oauth2.Config, authCode string) error {
	token, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return err
	}

	return saveToken(tokenFile, token)
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
