package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	cache      *EventsCache
	cacheMutex sync.RWMutex
	scraper    *MeetupScraper
	meetupURL  string
)

func main() {
	// Get environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	meetupURL = os.Getenv("MEETUP_ORG_URL")
	if meetupURL == "" {
		log.Fatal("MEETUP_ORG_URL environment variable is required")
	}

	refreshInterval := os.Getenv("REFRESH_INTERVAL_HOURS")
	if refreshInterval == "" {
		refreshInterval = "1"
	}

	log.Printf("Starting Meetup API Server...")
	log.Printf("Meetup URL: %s", meetupURL)
	log.Printf("Refresh interval: %s hour(s)", refreshInterval)

	// Initialize scraper and cache
	scraper = NewMeetupScraper(meetupURL)
	cache = &EventsCache{
		Upcoming:    []MeetupEvent{},
		Past:        []MeetupEvent{},
		LastUpdated: time.Now(),
	}

	// Initial cache load — retry so a transient scrape failure (e.g. Meetup's
	// persisted-query cache being cold) doesn't leave us serving an empty list.
	// After a bounded number of tries we start anyway (the port must open and the
	// periodic refresh keeps trying), but we try hard to have real data first.
	const maxInitialAttempts = 5
	for attempt := 1; attempt <= maxInitialAttempts; attempt++ {
		err := refreshCache()
		if err == nil {
			break
		}
		if attempt == maxInitialAttempts {
			log.Printf("WARNING: initial cache still degraded after %d attempts: %v — starting anyway, periodic refresh will keep retrying", attempt, err)
			break
		}
		wait := time.Duration(attempt*3) * time.Second
		log.Printf("Initial cache load attempt %d degraded (%v); retrying in %s...", attempt, err, wait)
		time.Sleep(wait)
	}

	// Setup cron for periodic refresh
	c := cron.New()
	cronExpr := "0 */" + refreshInterval + " * * *"
	c.AddFunc(cronExpr, func() {
		log.Println("Running scheduled cache refresh...")
		if err := refreshCache(); err != nil {
			log.Printf("Scheduled refresh failed: %v", err)
		}
	})
	c.Start()

	// Setup HTTP routes
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/upcoming", upcomingHandler)
	http.HandleFunc("/api/past", pastHandler)
	http.HandleFunc("/feed/upcoming", feedUpcomingHandler)
	http.HandleFunc("/feed/past", feedPastHandler)

	// Start server
	log.Printf("Server running on http://localhost:%s", port)
	log.Printf("API endpoints:")
	log.Printf("  - GET /health")
	log.Printf("  - GET /api/upcoming")
	log.Printf("  - GET /api/past")
	log.Printf("  - GET /feed/upcoming")
	log.Printf("  - GET /feed/past")

	if err := http.ListenAndServe(":"+port, addCacheHeaders(http.DefaultServeMux)); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func refreshCache() error {
	upcoming, past, err := scraper.ScrapeEvents()
	if err != nil {
		return err
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Defensive: never overwrite good data with an empty list. A dimension that
	// comes back empty is treated as a transient scrape failure for that dimension
	// (not "there are no events") — we keep whatever we already had.
	changed := false
	if len(upcoming) > 0 || len(cache.Upcoming) == 0 {
		cache.Upcoming = upcoming
		changed = true
	} else {
		log.Printf("WARNING: scrape returned 0 upcoming events; keeping %d cached", len(cache.Upcoming))
	}
	// Past events only ever accumulate, and we have two sources of differing
	// completeness (full GraphQL history vs. the ~10-item HTML fallback). So only
	// accept a past list at least as complete as what we already have: this keeps a
	// cold-cache fallback from clobbering a good full history, yet still upgrades an
	// earlier fallback once GraphQL recovers. (Also covers the empty case: 0 is only
	// accepted when the cache is itself empty.)
	if len(past) >= len(cache.Past) {
		cache.Past = past
		changed = true
	} else {
		log.Printf("WARNING: scrape returned %d past events but cache has %d; keeping cached (likely a degraded/fallback fetch)", len(past), len(cache.Past))
	}
	if changed {
		cache.LastUpdated = time.Now()
	}

	// Signal a degraded result so the initial-load retry loop can react. Upcoming
	// can legitimately be zero (nothing scheduled), but this established group
	// always has past events, so an empty past cache means the scrape isn't usable
	// yet (e.g. GraphQL cold-cache AND the HTML fallback came back empty).
	if len(cache.Past) == 0 {
		return fmt.Errorf("no past events available (scrape degraded)")
	}
	return nil
}

// Middleware to add cache headers
func addCacheHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=60")
		next.ServeHTTP(w, r)
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"name":    "Meetup API Server",
		"version": "1.0.0",
		"endpoints": map[string]interface{}{
			"health": "/health",
			"api": map[string]string{
				"upcoming": "/api/upcoming",
				"past":     "/api/past",
			},
			"rss": map[string]string{
				"upcoming": "/feed/upcoming",
				"past":     "/feed/past",
			},
		},
		"meetupUrl": meetupURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	lastUpdated := cache.LastUpdated
	cacheMutex.RUnlock()

	response := map[string]interface{}{
		"status":      "ok",
		"lastUpdated": lastUpdated.Format(time.RFC3339),
		"meetupUrl":   meetupURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func upcomingHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	events := cache.Upcoming
	lastUpdated := cache.LastUpdated
	cacheMutex.RUnlock()

	response := paginateEvents(r, events, lastUpdated)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func pastHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	events := cache.Past
	lastUpdated := cache.LastUpdated
	cacheMutex.RUnlock()

	response := paginateEvents(r, events, lastUpdated)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func paginateEvents(r *http.Request, allEvents []MeetupEvent, lastUpdated time.Time) map[string]interface{} {
	// Parse query parameters
	query := r.URL.Query()
	page := parseIntParam(query.Get("page"), 1)
	limit := parseIntParam(query.Get("limit"), 25)

	// Ensure valid values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 25 {
		limit = 25
	}

	totalEvents := len(allEvents)
	totalPages := (totalEvents + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Calculate slice boundaries
	start := (page - 1) * limit
	end := start + limit
	if start > totalEvents {
		start = totalEvents
	}
	if end > totalEvents {
		end = totalEvents
	}

	// Get paginated events
	paginatedEvents := allEvents[start:end]

	// Build pagination links
	baseURL := r.URL.Path
	links := map[string]interface{}{
		"self": buildPaginationURL(baseURL, page, limit),
	}

	if page > 1 {
		links["first"] = buildPaginationURL(baseURL, 1, limit)
		links["prev"] = buildPaginationURL(baseURL, page-1, limit)
	}

	if page < totalPages {
		links["next"] = buildPaginationURL(baseURL, page+1, limit)
		links["last"] = buildPaginationURL(baseURL, totalPages, limit)
	}

	response := map[string]interface{}{
		"events": paginatedEvents,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"totalEvents": totalEvents,
			"totalPages":  totalPages,
			"count":       len(paginatedEvents),
		},
		"links":       links,
		"lastUpdated": lastUpdated.Format(time.RFC3339),
	}

	return response
}

func parseIntParam(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func buildPaginationURL(basePath string, page int, limit int) string {
	return fmt.Sprintf("%s?page=%d&limit=%d", basePath, page, limit)
}

func feedUpcomingHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	events := cache.Upcoming
	cacheMutex.RUnlock()

	rss, err := GenerateRSSFeed(events, meetupURL, "upcoming")
	if err != nil {
		http.Error(w, "Failed to generate RSS feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(rss))
}

func feedPastHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	events := cache.Past
	cacheMutex.RUnlock()

	rss, err := GenerateRSSFeed(events, meetupURL, "past")
	if err != nil {
		http.Error(w, "Failed to generate RSS feed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Write([]byte(rss))
}
