package service

import (
	"calendar/domain"
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	calendarv3 "google.golang.org/api/calendar/v3"

	"google.golang.org/api/option"
)

type CalendarApp struct {
	service *calendarv3.Service
	id      string
}

func NewCalendarApp(ctx context.Context, client *http.Client) (*CalendarApp, error) {
	srv, err := calendarv3.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &CalendarApp{service: srv, id: "primary"}, nil
}

func (s *CalendarApp) CreateEvent(ctx context.Context, event *calendarv3.Event) error {
	res, err := s.service.Events.Insert(s.id, event).Do()
	if err != nil {
		return err
	}

	log.Debug().Interface("res", res).Send()

	return nil
}

func (s *CalendarApp) CreateEvents(ctx context.Context, events []*calendarv3.Event) error {
	groupID := uuid.New().String()
	for _, event := range events {
		if event.ExtendedProperties == nil {
			event.ExtendedProperties = &calendarv3.EventExtendedProperties{Private: make(map[string]string)}
		}
		if event.ExtendedProperties.Private == nil {
			event.ExtendedProperties.Private = make(map[string]string)
		}
		event.ExtendedProperties.Private["calendar-app-group-id"] = groupID
		err := s.CreateEvent(ctx, event)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *CalendarApp) DeleteEvent(ctx context.Context, id string) error {
	err := s.service.Events.Delete("primary", id).Do()
	if err != nil {
		return err
	}
	return nil
}

func (s *CalendarApp) FetchUpcomingEvents(ctx context.Context, filter domain.EventFilter) ([]*calendarv3.Event, error) {
	events, err := s.service.Events.List("primary").ShowDeleted(false).SingleEvents(filter.SingleEvents).
		TimeMin(filter.TimeMin).TimeMax(filter.TimeMax).
		MaxResults(filter.MaxResults).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, err
	}

	return events.Items, nil
}
