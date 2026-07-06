package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type MeetupScraper struct {
	meetupURL string
	client    *http.Client
}

func NewMeetupScraper(meetupURL string) *MeetupScraper {
	return &MeetupScraper{
		meetupURL: strings.TrimSuffix(meetupURL, "/"),
		client:    &http.Client{},
	}
}

func (s *MeetupScraper) ScrapeEvents() ([]MeetupEvent, []MeetupEvent, error) {
	log.Printf("Fetching events from %s...\n", s.meetupURL)

	// Fetch upcoming events from main page
	upcoming, err := s.fetchPage(s.meetupURL, "ACTIVE")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch upcoming events: %w", err)
	}

	// Fetch all past events with pagination
	past, err := s.fetchAllPastEvents()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch past events: %w", err)
	}

	log.Printf("Found %d upcoming events and %d past events\n", len(upcoming), len(past))

	return upcoming, past, nil
}

// fetchAllPastEvents returns past events, preferring the full GraphQL history but
// falling back to the HTML page so a cold Meetup persisted-query cache degrades to
// "the recent events" instead of zero.
func (s *MeetupScraper) fetchAllPastEvents() ([]MeetupEvent, error) {
	events, err := s.fetchAllPastEventsGraphQL()
	if err == nil && len(events) > 0 {
		return events, nil
	}

	// GraphQL failed (commonly PersistedQueryNotFound) or returned nothing. Scrape
	// the past-events HTML page, whose __NEXT_DATA__ embeds the most recent events
	// directly and does NOT depend on the persisted query. Fewer events, never zero.
	log.Printf("GraphQL past-events fetch unusable (err=%v, got=%d) — falling back to HTML scrape\n", err, len(events))
	htmlEvents, herr := s.fetchPage(s.meetupURL+"/events/?type=past", "PAST")
	if herr != nil {
		if err != nil {
			return nil, fmt.Errorf("past events unavailable: graphql (%v) and html fallback (%v) both failed", err, herr)
		}
		return nil, fmt.Errorf("past events html fallback failed: %w", herr)
	}
	log.Printf("HTML fallback returned %d past events\n", len(htmlEvents))
	return htmlEvents, nil
}

func (s *MeetupScraper) fetchAllPastEventsGraphQL() ([]MeetupEvent, error) {
	allEvents := []MeetupEvent{}
	cutoffYear := 2019

	// Extract urlname from meetupURL (e.g., "astoria-tech-meetup")
	urlname := strings.TrimPrefix(s.meetupURL, "https://www.meetup.com/")
	urlname = strings.TrimPrefix(urlname, "http://www.meetup.com/")
	urlname = strings.TrimSuffix(urlname, "/")

	log.Printf("Fetching past events using GraphQL API for org: %s\n", urlname)

	cursor := ""
	pageNum := 1
	maxPages := 100

	for pageNum <= maxPages {
		log.Printf("Fetching past events page %d (cursor: %s)...\n", pageNum, func() string {
			if cursor == "" {
				return "initial"
			}
			return cursor[:20] + "..."
		}())

		events, nextCursor, hasMore, err := s.fetchPastEventsGraphQL(urlname, cursor)
		if err != nil {
			log.Printf("Error fetching page %d: %v\n", pageNum, err)
			if len(allEvents) == 0 {
				return nil, fmt.Errorf("failed on first page of past events: %w", err)
			}
			break
		}

		if len(events) == 0 {
			log.Printf("No more events found on page %d\n", pageNum)
			break
		}

		// Check if we've reached events before cutoff year
		oldestOnPage := false
		for _, event := range events {
			if event.DateTime != "" && len(event.DateTime) >= 4 {
				year := 0
				fmt.Sscanf(event.DateTime[:4], "%d", &year)
				if year > 0 && year < cutoffYear {
					log.Printf("Reached events from year %d (before %d cutoff)\n", year, cutoffYear)
					oldestOnPage = true
					break
				}
			}
		}

		allEvents = append(allEvents, events...)
		log.Printf("Added %d events from page %d (total: %d)\n", len(events), pageNum, len(allEvents))

		if oldestOnPage {
			log.Printf("Stopping pagination - reached cutoff year %d\n", cutoffYear)
			break
		}

		if !hasMore || nextCursor == "" {
			log.Printf("No more pages available (hasMore=%v, cursor empty=%v)\n", hasMore, nextCursor == "")
			break
		}

		cursor = nextCursor
		pageNum++

		// Rate limiting: wait 1 second between requests to be polite
		time.Sleep(1 * time.Second)
	}

	if pageNum > maxPages {
		log.Printf("Reached maximum page limit (%d)\n", maxPages)
	}

	// Sort events by date (newest first)
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].DateTime > allEvents[j].DateTime
	})

	log.Printf("Total past events fetched: %d\n", len(allEvents))
	return allEvents, nil
}

type GraphQLRequest struct {
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
	Extensions    map[string]interface{} `json:"extensions"`
}

type GraphQLResponse struct {
	// Meetup's GraphQL uses Automatic Persisted Queries: we send only the query's
	// sha256 hash. When Meetup's edge cache doesn't have that hash registered it
	// replies HTTP 200 with a non-empty `errors` array (typically
	// "PersistedQueryNotFound") and null data. We must surface that as an error —
	// otherwise the empty `edges` below look like a legitimate "no past events".
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
	Data struct {
		GroupByUrlname struct {
			Events struct {
				TotalCount int `json:"totalCount"`
				PageInfo   struct {
					EndCursor   string `json:"endCursor"`
					HasNextPage bool   `json:"hasNextPage"`
				} `json:"pageInfo"`
				Edges []struct {
					Node struct {
						ID          string `json:"id"`
						Title       string `json:"title"`
						Description string `json:"description"`
						DateTime    string `json:"dateTime"`
						EndTime     string `json:"endTime"`
						EventURL    string `json:"eventUrl"`
						Status      string `json:"status"`
						Going       struct {
							TotalCount int `json:"totalCount"`
						} `json:"going"`
						Venue *struct {
							Name    string `json:"name"`
							Address string `json:"address"`
							City    string `json:"city"`
							State   string `json:"state"`
						} `json:"venue"`
						FeaturedEventPhoto *struct {
							HighResUrl string `json:"highResUrl"`
						} `json:"featuredEventPhoto"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"events"`
		} `json:"groupByUrlname"`
	} `json:"data"`
}

func (s *MeetupScraper) fetchPastEventsGraphQL(urlname string, cursor string) ([]MeetupEvent, string, bool, error) {
	gqlURL := "https://www.meetup.com/gql2"

	// Build request payload
	variables := map[string]interface{}{
		"urlname":        urlname,
		"beforeDateTime": time.Now().UTC().Format(time.RFC3339),
	}

	if cursor != "" {
		variables["after"] = cursor
	}

	payload := GraphQLRequest{
		OperationName: "getPastGroupEvents",
		Variables:     variables,
		Extensions: map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": "84d621b514d4bfad36d9b37d78f469ee558b01ebe97ba9fb9183fe958b2ad1f1",
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", gqlURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("apollographql-client-name", "nextjs-web")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, "", false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	// A 200 can still carry GraphQL-level errors (e.g. PersistedQueryNotFound when
	// the query hash is cold in Meetup's cache). Treat that as a hard error so the
	// empty edge list isn't mistaken for a real "no more events" result.
	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, 0, len(gqlResp.Errors))
		for _, e := range gqlResp.Errors {
			msgs = append(msgs, e.Message)
		}
		return nil, "", false, fmt.Errorf("graphql error: %s", strings.Join(msgs, "; "))
	}

	// Extract events
	events := []MeetupEvent{}
	for _, edge := range gqlResp.Data.GroupByUrlname.Events.Edges {
		node := edge.Node

		// Only include PAST events (filter since the query may return all events)
		if node.Status != "PAST" {
			continue
		}

		event := MeetupEvent{
			ID:          node.ID,
			Title:       node.Title,
			Description: stripHTML(node.Description),
			DateTime:    node.DateTime,
			EndTime:     node.EndTime,
			EventURL:    node.EventURL,
			Going:       node.Going.TotalCount,
			Status:      node.Status,
		}

		if node.Venue != nil {
			event.Location = &Location{
				Name:    node.Venue.Name,
				Address: node.Venue.Address,
				City:    node.Venue.City,
				State:   node.Venue.State,
			}
		}

		if node.FeaturedEventPhoto != nil {
			event.PhotoURL = node.FeaturedEventPhoto.HighResUrl
		}

		events = append(events, event)
	}

	nextCursor := gqlResp.Data.GroupByUrlname.Events.PageInfo.EndCursor
	hasMore := gqlResp.Data.GroupByUrlname.Events.PageInfo.HasNextPage

	return events, nextCursor, hasMore, nil
}

func (s *MeetupScraper) fetchPage(url string, status string) ([]MeetupEvent, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the __NEXT_DATA__ script tag
	var nextDataJSON string
	doc.Find("script#__NEXT_DATA__").Each(func(i int, s *goquery.Selection) {
		nextDataJSON = s.Text()
	})

	if nextDataJSON == "" {
		return nil, fmt.Errorf("could not find __NEXT_DATA__ in the page")
	}

	// Parse the JSON
	var nextData map[string]interface{}
	if err := json.Unmarshal([]byte(nextDataJSON), &nextData); err != nil {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__: %w", err)
	}

	// Navigate to Apollo state
	props, ok := nextData["props"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not find props in __NEXT_DATA__")
	}

	pageProps, ok := props["pageProps"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not find pageProps")
	}

	apolloState, ok := pageProps["__APOLLO_STATE__"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("could not find Apollo state in the page data")
	}

	// Extract events
	events := s.extractEventsFromApolloState(apolloState, status)

	return events, nil
}

func (s *MeetupScraper) extractEventsFromApolloState(apolloState map[string]interface{}, status string) []MeetupEvent {
	events := []MeetupEvent{}

	for key, value := range apolloState {
		if !strings.HasPrefix(key, "Event:") {
			continue
		}

		eventData, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		// Filter by status
		eventStatus, _ := eventData["status"].(string)
		if eventStatus != status {
			continue
		}

		event := MeetupEvent{
			Status: status,
		}

		// Extract basic fields
		if id, ok := eventData["id"].(string); ok {
			event.ID = id
		} else {
			event.ID = strings.TrimPrefix(key, "Event:")
		}

		if title, ok := eventData["title"].(string); ok {
			event.Title = title
		} else {
			event.Title = "Untitled Event"
		}

		if desc, ok := eventData["description"].(string); ok {
			event.Description = stripHTML(desc)
		}

		if dateTime, ok := eventData["dateTime"].(string); ok {
			event.DateTime = dateTime
		} else if startDate, ok := eventData["startDate"].(string); ok {
			event.DateTime = startDate
		}

		if endTime, ok := eventData["endTime"].(string); ok {
			event.EndTime = endTime
		} else if endDate, ok := eventData["endDate"].(string); ok {
			event.EndTime = endDate
		}

		// Extract venue/location
		if venue, ok := eventData["venue"].(map[string]interface{}); ok {
			event.Location = &Location{}

			// Check for __ref (reference to another object)
			if ref, ok := venue["__ref"].(string); ok {
				if venueData, ok := apolloState[ref].(map[string]interface{}); ok {
					venue = venueData
				}
			}

			if name, ok := venue["name"].(string); ok {
				event.Location.Name = name
			}
			if address, ok := venue["address"].(string); ok {
				event.Location.Address = address
			}
			if city, ok := venue["city"].(string); ok {
				event.Location.City = city
			}
			if state, ok := venue["state"].(string); ok {
				event.Location.State = state
			}
		}

		// Extract going count
		if going, ok := eventData["going"].(float64); ok {
			event.Going = int(going)
		} else if goingObj, ok := eventData["going"].(map[string]interface{}); ok {
			if totalCount, ok := goingObj["totalCount"].(float64); ok {
				event.Going = int(totalCount)
			}
		}

		// Set event URL
		if eventURL, ok := eventData["eventUrl"].(string); ok {
			event.EventURL = eventURL
		} else {
			event.EventURL = fmt.Sprintf("%s/events/%s", s.meetupURL, event.ID)
		}

		events = append(events, event)
	}

	// Sort events
	sort.Slice(events, func(i, j int) bool {
		if status == "ACTIVE" {
			return events[i].DateTime < events[j].DateTime
		}
		return events[i].DateTime > events[j].DateTime
	})

	return events
}

func stripHTML(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return html
	}
	return strings.TrimSpace(doc.Text())
}
