package calendar

// import (
// 	"context"
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"calendar/internal/model"

// 	google_calendar "google.golang.org/api/calendar/v3"
// )

// const (
// 	AppCalendarSummary = "Calendar App"
// 	ExtType            = "calendar-app-type"
// 	ExtTypeParent      = "parent"
// 	ExtTypeSubevent    = "subevent"
// 	ExtParentID        = "calendar-app-parent-id"
// 	ExtDayOffset       = "calendar-app-day-offset"
// 	ExtSource          = "calendar-app-source"
// 	ExtSourceValue     = "calendar-app"
// )

// // Service operates on the app calendar using the given Google Calendar API client.
// type Service struct {
// 	svc   *google_calendar.Service
// 	calID string
// }

// // NewService creates a calendar service. It ensures the app calendar exists and uses it.
// func NewService(ctx context.Context, svc *google_calendar.Service) (*Service, error) {
// 	calID, err := ensureAppCalendar(ctx, svc)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &Service{svc: svc, calID: calID}, nil
// }

// func ensureAppCalendar(ctx context.Context, svc *google_calendar.Service) (string, error) {
// 	list, err := svc.CalendarList.List().Do()
// 	if err != nil {
// 		return "", err
// 	}
// 	for _, item := range list.Items {
// 		if item.Summary == AppCalendarSummary {
// 			return item.Id, nil
// 		}
// 	}
// 	cal := &google_calendar.Calendar{
// 		Summary:  AppCalendarSummary,
// 		TimeZone: "UTC",
// 	}
// 	created, err := svc.Calendars.Insert(cal).Context(ctx).Do()
// 	if err != nil {
// 		return "", err
// 	}
// 	return created.Id, nil
// }

// // CreateEvent creates a parent event on the app calendar.
// func (s *Service) CreateEvent(ctx context.Context, req *model.CreateEventRequest) (*model.Event, error) {
// 	ev := &google_calendar.Event{
// 		Summary:     req.Title,
// 		Description: req.Description,
// 		Start: &google_calendar.EventDateTime{
// 			Date:     req.Date,
// 			TimeZone: req.Timezone,
// 		},
// 		End: &google_calendar.EventDateTime{
// 			Date:     req.Date,
// 			TimeZone: req.Timezone,
// 		},
// 		ExtendedProperties: &google_calendar.EventExtendedProperties{
// 			Private: map[string]string{
// 				ExtType:   ExtTypeParent,
// 				ExtSource: ExtSourceValue,
// 			},
// 		},
// 	}
// 	created, err := s.svc.Events.Insert(s.calID, ev).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return googleEventToModel(created), nil
// }

// // ListEvents lists parent events from the app calendar (optionally filtered by timeMin/timeMax).
// func (s *Service) ListEvents(ctx context.Context, timeMin, timeMax string) ([]model.Event, error) {
// 	call := s.svc.Events.List(s.calID).SingleEvents(true).Context(ctx)
// 	if timeMin != "" {
// 		call = call.TimeMin(timeMin + "T00:00:00Z")
// 	}
// 	if timeMax != "" {
// 		call = call.TimeMax(timeMax + "T23:59:59Z")
// 	}
// 	events, err := call.Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	var out []model.Event
// 	for _, e := range events.Items {
// 		if e.ExtendedProperties != nil && e.ExtendedProperties.Private[ExtType] == ExtTypeParent {
// 			out = append(out, *googleEventToModel(e))
// 		}
// 	}
// 	return out, nil
// }

// // GetEvent returns one parent event by ID with its sub-events.
// func (s *Service) GetEvent(ctx context.Context, eventID string) (*model.EventWithSubEvents, error) {
// 	ev, err := s.svc.Events.Get(s.calID, eventID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if ev.ExtendedProperties == nil || ev.ExtendedProperties.Private[ExtType] != ExtTypeParent {
// 		return nil, fmt.Errorf("event not found or not a parent event")
// 	}
// 	parent := googleEventToModel(ev)
// 	subs, err := s.listSubEventsForParent(ctx, eventID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &model.EventWithSubEvents{Event: *parent, SubEvents: subs}, nil
// }

// // UpdateEvent updates a parent event.
// func (s *Service) UpdateEvent(ctx context.Context, eventID string, req *model.UpdateEventRequest) (*model.Event, error) {
// 	ev, err := s.svc.Events.Get(s.calID, eventID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if ev.ExtendedProperties == nil || ev.ExtendedProperties.Private[ExtType] != ExtTypeParent {
// 		return nil, fmt.Errorf("event not found or not a parent event")
// 	}
// 	if req.Title != nil {
// 		ev.Summary = *req.Title
// 	}
// 	if req.Date != nil {
// 		ev.Start.Date = *req.Date
// 		ev.End.Date = *req.Date
// 	}
// 	if req.Timezone != nil {
// 		ev.Start.TimeZone = *req.Timezone
// 		ev.End.TimeZone = *req.Timezone
// 	}
// 	if req.Description != nil {
// 		ev.Description = *req.Description
// 	}
// 	updated, err := s.svc.Events.Update(s.calID, eventID, ev).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return googleEventToModel(updated), nil
// }

// // DeleteEvent deletes a parent event and all its sub-events.
// func (s *Service) DeleteEvent(ctx context.Context, eventID string) error {
// 	subs, err := s.listSubEventsForParent(ctx, eventID)
// 	if err != nil {
// 		return err
// 	}
// 	for _, sub := range subs {
// 		_ = s.svc.Events.Delete(s.calID, sub.ID).Context(ctx).Do()
// 	}
// 	return s.svc.Events.Delete(s.calID, eventID).Context(ctx).Do()
// }

// // CreateSubEvent creates a sub-event (reminder) for a parent event.
// func (s *Service) CreateSubEvent(ctx context.Context, parentID string, req *model.CreateSubEventRequest) (*model.SubEvent, error) {
// 	parent, err := s.svc.Events.Get(s.calID, parentID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if parent.ExtendedProperties == nil || parent.ExtendedProperties.Private[ExtType] != ExtTypeParent {
// 		return nil, fmt.Errorf("parent event not found")
// 	}
// 	parentDate, err := time.Parse("2006-01-02", parent.Start.Date)
// 	if err != nil {
// 		return nil, err
// 	}
// 	occurrenceDate := parentDate.AddDate(0, 0, req.DayOffset).Format("2006-01-02")
// 	tz := parent.Start.TimeZone
// 	if tz == "" {
// 		tz = "UTC"
// 	}
// 	ev := &google_calendar.Event{
// 		Summary:     parent.Summary + ": " + req.Label,
// 		Description: req.Description,
// 		Start: &google_calendar.EventDateTime{
// 			Date:     occurrenceDate,
// 			TimeZone: tz,
// 		},
// 		End: &google_calendar.EventDateTime{
// 			Date:     occurrenceDate,
// 			TimeZone: tz,
// 		},
// 		ExtendedProperties: &google_calendar.EventExtendedProperties{
// 			Private: map[string]string{
// 				ExtType:      ExtTypeSubevent,
// 				ExtParentID:  parentID,
// 				ExtDayOffset: strconv.Itoa(req.DayOffset),
// 				ExtSource:    ExtSourceValue,
// 			},
// 		},
// 	}
// 	created, err := s.svc.Events.Insert(s.calID, ev).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &model.SubEvent{
// 		ID:             created.Id,
// 		EventID:        parentID,
// 		Label:          req.Label,
// 		DayOffset:      req.DayOffset,
// 		Description:    req.Description,
// 		OccurrenceDate: occurrenceDate,
// 	}, nil
// }

// // ListSubEvents returns sub-events for a parent event.
// func (s *Service) ListSubEvents(ctx context.Context, parentID string) ([]model.SubEvent, error) {
// 	return s.listSubEventsForParent(ctx, parentID)
// }

// func (s *Service) listSubEventsForParent(ctx context.Context, parentID string) ([]model.SubEvent, error) {
// 	parent, err := s.svc.Events.Get(s.calID, parentID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	prefix := parent.Summary + ": "
// 	events, err := s.svc.Events.List(s.calID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	var out []model.SubEvent
// 	for _, e := range events.Items {
// 		if e.ExtendedProperties == nil || e.ExtendedProperties.Private[ExtType] != ExtTypeSubevent {
// 			continue
// 		}
// 		if e.ExtendedProperties.Private[ExtParentID] != parentID {
// 			continue
// 		}
// 		offset, _ := strconv.Atoi(e.ExtendedProperties.Private[ExtDayOffset])
// 		label := e.Summary
// 		if strings.HasPrefix(e.Summary, prefix) {
// 			label = strings.TrimPrefix(e.Summary, prefix)
// 		}
// 		out = append(out, model.SubEvent{
// 			ID:             e.Id,
// 			EventID:        parentID,
// 			Label:          label,
// 			DayOffset:      offset,
// 			Description:    e.Description,
// 			OccurrenceDate: e.Start.Date,
// 		})
// 	}
// 	return out, nil
// }

// // UpdateSubEvent updates a sub-event.
// func (s *Service) UpdateSubEvent(ctx context.Context, parentID, subID string, req *model.UpdateSubEventRequest) (*model.SubEvent, error) {
// 	sub, err := s.svc.Events.Get(s.calID, subID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	if sub.ExtendedProperties == nil || sub.ExtendedProperties.Private[ExtParentID] != parentID {
// 		return nil, fmt.Errorf("sub-event not found")
// 	}
// 	parent, err := s.svc.Events.Get(s.calID, parentID).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	offset, _ := strconv.Atoi(sub.ExtendedProperties.Private[ExtDayOffset])
// 	if req.Label != nil {
// 		sub.Summary = parent.Summary + ": " + *req.Label
// 	}
// 	if req.DayOffset != nil {
// 		offset = *req.DayOffset
// 		parentDate, _ := time.Parse("2006-01-02", parent.Start.Date)
// 		occurrenceDate := parentDate.AddDate(0, 0, offset).Format("2006-01-02")
// 		sub.Start.Date = occurrenceDate
// 		sub.End.Date = occurrenceDate
// 		sub.ExtendedProperties.Private[ExtDayOffset] = strconv.Itoa(offset)
// 	}
// 	if req.Description != nil {
// 		sub.Description = *req.Description
// 	}
// 	updated, err := s.svc.Events.Update(s.calID, subID, sub).Context(ctx).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	label := updated.Summary
// 	if len(parent.Summary) < len(updated.Summary) && updated.Summary[:len(parent.Summary)+2] == parent.Summary+": " {
// 		label = updated.Summary[len(parent.Summary)+2:]
// 	}
// 	return &model.SubEvent{
// 		ID:             updated.Id,
// 		EventID:        parentID,
// 		Label:          label,
// 		DayOffset:      offset,
// 		Description:    updated.Description,
// 		OccurrenceDate: updated.Start.Date,
// 	}, nil
// }

// // DeleteSubEvent deletes a sub-event.
// func (s *Service) DeleteSubEvent(ctx context.Context, parentID, subID string) error {
// 	sub, err := s.svc.Events.Get(s.calID, subID).Context(ctx).Do()
// 	if err != nil {
// 		return err
// 	}
// 	if sub.ExtendedProperties == nil || sub.ExtendedProperties.Private[ExtParentID] != parentID {
// 		return fmt.Errorf("sub-event not found")
// 	}
// 	return s.svc.Events.Delete(s.calID, subID).Context(ctx).Do()
// }

// func googleEventToModel(e *google_calendar.Event) *model.Event {
// 	ev := &model.Event{
// 		ID:          e.Id,
// 		Title:       e.Summary,
// 		Date:        e.Start.Date,
// 		Timezone:    e.Start.TimeZone,
// 		Description: e.Description,
// 	}
// 	if e.Created != "" {
// 		ev.CreatedAt, _ = time.Parse(time.RFC3339, e.Created)
// 	}
// 	if e.Updated != "" {
// 		ev.UpdatedAt, _ = time.Parse(time.RFC3339, e.Updated)
// 	}
// 	return ev
// }
