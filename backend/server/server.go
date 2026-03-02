package server

import (
	"calendar/domain"
	"calendar/service"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

type CalendarServer struct {
	config   *oauth2.Config
	calendar *service.CalendarApp
	token    string
}

func NewCalendarServer(config *oauth2.Config) *CalendarServer {
	s := &CalendarServer{config: config}
	// lazy init
	client, err := getClient(config)
	if err == nil {
		calendar, err := service.NewCalendarApp(context.TODO(), client)
		if err != nil {
			log.Err(err).Msg("failed to create calendar app at startup, requires authentication")
		}
		s.calendar = calendar
	}

	return s
}

func (s *CalendarServer) GetAuthURLHandler(w http.ResponseWriter, r *http.Request) {
	url := getTokenURL(s.config)
	writeJSON(w, http.StatusOK, map[string]interface{}{"url": url})
}

func (s *CalendarServer) AuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	authCode := r.URL.Query().Get("code")
	if authCode == "" {
		http.Error(w, fmt.Errorf("missing auth code").Error(), http.StatusInternalServerError)
		return
	}

	err := saveTokenFromWeb(s.config, authCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client, err := getClient(s.config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	calendar, err := service.NewCalendarApp(r.Context(), client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.calendar = calendar

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *CalendarServer) ListEventsHandler(w http.ResponseWriter, r *http.Request) {
	if s.calendar == nil {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{"error": "authentication failed"})
		return
	}

	events, err := s.calendar.FetchUpcomingEvents(r.Context(), domain.EventFilter{
		TimeMin:      time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
		TimeMax:      time.Now().AddDate(1, 0, 0).Format(time.RFC3339),
		SingleEvents: true,
		MaxResults:   50,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": events})
}

func (s *CalendarServer) DeleteEventHandler(w http.ResponseWriter, r *http.Request) {
	if s.calendar == nil {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{"error": "authentication failed"})
		return
	}

	eventID := r.URL.Query().Get("eventId")
	err := s.calendar.DeleteEvent(r.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "Event deleted"})
}

func (s *CalendarServer) CreateEventsHandler(w http.ResponseWriter, r *http.Request) {
	if s.calendar == nil {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{"error": "authentication failed"})
		return
	}
	var events []*calendar.Event
	err := json.NewDecoder(r.Body).Decode(&events)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.calendar.CreateEvents(r.Context(), events)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "Events created"})
}

func writeJSON(w http.ResponseWriter, i int, v map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(i)
	_ = json.NewEncoder(w).Encode(v)
}
