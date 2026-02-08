package main

import "time"

type MeetupEvent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DateTime    string    `json:"dateTime"`
	EndTime     string    `json:"endTime,omitempty"`
	Location    *Location `json:"location,omitempty"`
	EventURL    string    `json:"eventUrl"`
	Going       int       `json:"going"`
	Status      string    `json:"status"`
	PhotoURL    string    `json:"photoUrl,omitempty"`
}

type Location struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
}

type EventsCache struct {
	Upcoming    []MeetupEvent
	Past        []MeetupEvent
	LastUpdated time.Time
}
