package model

import "time"

// Event is the API representation of a parent event (stored as one Google Calendar event).
type Event struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Date        string    `json:"date"` // YYYY-MM-DD
	Timezone    string    `json:"timezone"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// SubEvent is the API representation of a reminder/milestone (stored as a separate Google Calendar event).
type SubEvent struct {
	ID             string `json:"id"`
	EventID        string `json:"event_id"`
	Label          string `json:"label"`
	DayOffset      int    `json:"day_offset"`
	Description    string `json:"description,omitempty"`
	OccurrenceDate string `json:"occurrence_date"` // YYYY-MM-DD (parent date + offset)
}

// CreateEventRequest is the request body for POST /events.
type CreateEventRequest struct {
	Title       string `json:"title"`
	Date        string `json:"date"` // YYYY-MM-DD
	Timezone    string `json:"timezone"`
	Description string `json:"description,omitempty"`
}

// UpdateEventRequest is the request body for PUT /events/:id.
type UpdateEventRequest struct {
	Title       *string `json:"title,omitempty"`
	Date        *string `json:"date,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
	Description *string `json:"description,omitempty"`
}

// CreateSubEventRequest is the request body for POST /events/:id/subevents.
type CreateSubEventRequest struct {
	Label       string `json:"label"`
	DayOffset   int    `json:"day_offset"`
	Description string `json:"description,omitempty"`
}

// UpdateSubEventRequest is the request body for PATCH /events/:eventId/subevents/:subId.
type UpdateSubEventRequest struct {
	Label       *string `json:"label,omitempty"`
	DayOffset   *int    `json:"day_offset,omitempty"`
	Description *string `json:"description,omitempty"`
}

// EventWithSubEvents is the response for GET /events/:id.
type EventWithSubEvents struct {
	Event     Event      `json:"event"`
	SubEvents []SubEvent `json:"sub_events"`
}
