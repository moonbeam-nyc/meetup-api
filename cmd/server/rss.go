package main

import (
	"time"

	"github.com/gorilla/feeds"
)

func GenerateRSSFeed(events []MeetupEvent, meetupURL, feedType string) (string, error) {
	feed := &feeds.Feed{
		Title:       "Meetup Events - " + feedType,
		Link:        &feeds.Link{Href: meetupURL},
		Description: "Meetup.com events feed",
		Created:     time.Now(),
	}

	for _, event := range events {
		item := &feeds.Item{
			Title:       event.Title,
			Link:        &feeds.Link{Href: event.EventURL},
			Description: event.Description,
			Created:     time.Now(),
		}

		// Parse dateTime if available
		if event.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, event.DateTime); err == nil {
				item.Created = t
			}
		}

		// Add location to description if available
		if event.Location != nil {
			loc := ""
			if event.Location.Name != "" {
				loc += event.Location.Name
			}
			if event.Location.City != "" {
				if loc != "" {
					loc += ", "
				}
				loc += event.Location.City
			}
			if event.Location.State != "" {
				if loc != "" {
					loc += ", "
				}
				loc += event.Location.State
			}
			if loc != "" {
				item.Description += "\n\nLocation: " + loc
			}
		}

		feed.Items = append(feed.Items, item)
	}

	return feed.ToRss()
}
