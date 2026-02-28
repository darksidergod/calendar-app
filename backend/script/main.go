// Package main runs the standalone Google Calendar script: reads date and workshop from stdin,
// creates workshop reminder events on the user's primary calendar (using token.json and credentials.json).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	google_calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatal().AnErr("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatal().AnErr("Unable to retrieve token from web: %v", err)
	}
	return tok
}

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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal().AnErr("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	ctx := context.Background()
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatal().AnErr("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, google_calendar.CalendarScope)
	if err != nil {
		log.Fatal().AnErr("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	calendar, err := google_calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatal().AnErr("Unable to retrieve Calendar client: %v", err)
	}

	list, err := calendar.CalendarList.List().Do()
	if err != nil {
		log.Fatal().AnErr("unable to list calendar %v", err)
	}

	var ids []string
	for _, item := range list.Items {
		ids = append(ids, item.Id)
	}
	log.Debug().Interface("ids", ids).Send()

	var dateString string
	if _, err := fmt.Scan(&dateString); err != nil {
		log.Err(err).Msg("failed to scan date")
		return
	}

	parsed, err := time.Parse("02-01-2006", dateString)
	if err != nil {
		log.Err(err).Msg("failed to parse date")
		return
	}

	var workshop string
	if _, err := fmt.Scan(&workshop); err != nil {
		log.Err(err).Msg("failed to scan workshop name")
		return
	}

	reminders := map[string]int{
		"Get Modules":              -7,
		"Update PMT":               -7,
		"Module 1st Read":          -5,
		"Module Prep and TLM Prep": -3,
		"Rehearse":                 -1,
		"Pictures":                 0,
		"Pre and Post Check":       0,
		"PMT Update":               1,
		"Pictures Upload":          1,
		"Pre and Post Analysis":    1,
	}

	for topic, offset := range reminders {
		occursOn := parsed
		event := &google_calendar.Event{
			Summary: workshop + " Workshop:" + topic,
			Start: &google_calendar.EventDateTime{
				Date:     occursOn.AddDate(0, 0, offset).Format("2006-01-02"),
				TimeZone: "Asia/Kolkata",
			},
			End: &google_calendar.EventDateTime{
				Date:     occursOn.AddDate(0, 0, offset).Format("2006-01-02"),
				TimeZone: "Asia/Kolkata",
			},
		}

		_, err := calendar.Events.Insert("primary", event).Do()
		if err != nil {
			log.Err(err).Send()
			return
		}
	}
}
