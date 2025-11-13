<p align="center">
  <img src="static/images/meetup-api.png" alt="Meetup API" width="256">
</p>

# Meetup API Server

A lightweight Go server that scrapes Meetup.com events and serves them via REST API and RSS feeds.

## Features

- Scrapes past and upcoming events from any Meetup.com group
- Serves events via REST API endpoints
- Generates RSS feeds for both upcoming and past events
- Automatic hourly refresh (configurable)
- Docker containerized with Docker Compose
- Tiny Docker image: **6MB** (26x smaller than Node.js version)
- Fast startup and low memory footprint

## Prerequisites

- Docker and Docker Compose
- Make (optional, for easier workflow)

## Quick Start

1. Edit `compose.yaml` and set your `MEETUP_ORG_URL`
2. Run `docker compose up` to start the server
3. Visit http://localhost:3000

Or using Make:
```bash
make dev
```

## API Endpoints

### Health Check
```
GET /health
```

Returns server status and last cache update time.

**Example Response:**
```json
{
  "status": "ok",                           // Server health status
  "lastUpdated": "2025-11-13T10:30:00Z"    // Last time events were refreshed
}
```

### Upcoming Events
```
GET /api/upcoming
```

Returns JSON array of upcoming events.

**Example Response:**
```json
[
  {
    "id": "abc123def",                      // Unique event identifier
    "title": "Monthly Tech Meetup",         // Event title
    "description": "Join us for an evening of tech talks and networking...",  // Event description (HTML stripped)
    "dateTime": "2025-11-20T18:00:00Z",    // Event start date/time (ISO 8601)
    "endTime": "2025-11-20T21:00:00Z",     // Event end date/time (optional)
    "location": {
      "name": "Tech Hub Downtown",          // Venue name
      "address": "123 Main St",             // Street address
      "city": "San Francisco",              // City
      "state": "CA"                         // State/province
    },
    "eventUrl": "https://www.meetup.com/your-group/events/abc123def/",  // Direct link to event
    "going": 42,                            // Number of RSVPs
    "status": "ACTIVE"                      // Event status (ACTIVE or PAST)
  }
]
```

### Past Events
```
GET /api/past
```

Returns JSON array of past events with the same structure as upcoming events.

**Example Response:**
```json
[
  {
    "id": "xyz789ghi",
    "title": "October Networking Event",
    "description": "Great evening of networking and collaboration...",
    "dateTime": "2025-10-15T18:00:00Z",
    "endTime": "2025-10-15T21:00:00Z",
    "location": {
      "name": "Innovation Center",
      "address": "456 Tech Blvd",
      "city": "San Francisco",
      "state": "CA"
    },
    "eventUrl": "https://www.meetup.com/your-group/events/xyz789ghi/",
    "going": 38,
    "status": "PAST"
  }
]
```

### RSS Feeds
```
GET /feed/upcoming
GET /feed/past
```

Returns RSS 2.0 feeds for upcoming and past events. Compatible with all standard RSS readers.

## Documentation

### Make Commands

The project includes a Makefile for common operations:

```bash
# Development
make dev              # Start server in Docker Compose
make dev-build        # Build and start server
make dev-logs         # View Docker logs
make dev-down         # Stop Docker services

# Testing
make test-server      # Run integration tests in Docker

# Production Docker
make build            # Build production Docker image
make push             # Push to GitHub Container Registry
make docker-all       # Build and push

# Kubernetes
make k8s-apply        # Deploy to Kubernetes cluster
make k8s-status       # Check deployment status
make k8s-logs         # Follow logs
make k8s-restart      # Rolling restart

# Utilities
make go-fmt           # Format Go code
make help             # Show all available targets
```

### Docker Compose Commands

Manual Docker Compose usage:

```bash
# Build and run
docker compose build
docker compose up -d

# View logs
docker compose logs -f

# Stop and remove containers
docker compose down

# Rebuild from scratch
docker compose build --no-cache
docker compose up -d
```

## Configuration

All configuration is in `compose.yaml`. Edit the environment variables:

- `MEETUP_ORG_URL`: Meetup.com group URL (required) - **Edit this for your meetup**
- `PORT`: Server port (default: 3000)
- `REFRESH_INTERVAL_HOURS`: Cache refresh interval (default: 1)

Example:
```yaml
environment:
  - MEETUP_ORG_URL=https://www.meetup.com/your-meetup-name/
  - PORT=3000
  - REFRESH_INTERVAL_HOURS=1
```

## Development

### Docker Development

All development is designed to run in Docker. Configuration is in `compose.yaml`.

```bash
# Start server in Docker
make dev

# View logs
make dev-logs

# Stop server
make dev-down

# Rebuild and start
make dev-build
```

To change the Meetup URL, edit the `MEETUP_ORG_URL` in `compose.yaml`.

### Testing

The project includes an integration test script that runs in Docker:

```bash
# Run tests (no setup needed - uses Docker Compose)
make test-server
```

The test script:
1. Starts the server in Docker Compose (detached mode)
2. Waits up to 10 seconds for the server to be ready
3. Tests all API endpoints (/, /health, /api/upcoming, /api/past)
4. Tests all RSS endpoints (/feed/upcoming, /feed/past)
5. Validates HTTP 200 responses and content types
6. Stops Docker Compose and cleans up

**Requirements:** Docker and curl

### Code Quality

```bash
# Format Go code
make go-fmt
```

## How It Works

1. The server fetches the Meetup.com group page
2. Extracts the `__NEXT_DATA__` script tag containing GraphQL data
3. Parses the Apollo cache state to extract event information
4. Caches events in memory
5. Refreshes cache every hour (configurable)
6. Serves events via REST API and RSS feeds

## Architecture

- `cmd/server/main.go`: HTTP server, route handlers, and caching logic
- `cmd/server/scraper.go`: Meetup.com scraping with goquery
- `cmd/server/rss.go`: RSS feed generation
- `cmd/server/types.go`: Go type definitions
- `Dockerfile`: Multi-stage build producing 6MB image
- `k8s/`: Kubernetes manifests for deployment

## Kubernetes Deployment

This project is designed to run in a Kubernetes cluster and uses GitHub Container Registry for image storage.

### Prerequisites

- kubectl configured with cluster access
- Docker authenticated with GitHub Container Registry

### Deployment Steps

1. Update the ConfigMap in `k8s/deployment.yaml` with your Meetup URL
2. Update the Ingress in `k8s/ingress.yaml` with your domain
3. Build and push the image:
   ```bash
   make docker-all
   ```
4. Deploy to Kubernetes:
   ```bash
   make k8s-apply
   ```
5. Check deployment status:
   ```bash
   make k8s-status
   ```

See `CLAUDE.md` for detailed deployment documentation.

## License

MIT
