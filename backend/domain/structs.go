package domain

import "time"

type Entry struct {
	Name       string     // prefix will be added to all entries related to this entry
	SubEntries []SubEntry // all associated calendar sub-entries
}

type SubEntry struct {
	Name      string
	Type      string
	StartTime time.Time
	EndTime   time.Time
	Timezone  string
}

type EventFilter struct {
	TimeMin      string
	TimeMax      string
	Summary      string
	Description  string
	EventType    string
	SingleEvents bool
	MaxResults   int64
}
