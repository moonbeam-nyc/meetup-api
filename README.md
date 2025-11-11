# Meetup API Server

A TypeScript Node.js server that scrapes Meetup.com events and serves them via REST API and RSS feeds.

## Features

- Scrapes past and upcoming events from any Meetup.com group
- Serves events via REST API endpoints
- Generates RSS feeds for both upcoming and past events
- Automatic hourly refresh (configurable)
- Docker containerized with Docker Compose
- TypeScript for type safety

## Quick Start

1. Edit `compose.yaml` and set your `MEETUP_ORG_URL`
2. Run `make dev` to start the server
3. Visit http://localhost:3000

### Prerequisites

- Docker and Docker Compose
- Make (optional, for easier workflow)

### Using Make (Recommended)

```bash
# Local development (runs in Docker)
make dev              # Start server in Docker Compose
make dev-build        # Build and start server
make dev-logs         # View Docker logs
make dev-down         # Stop Docker services

# Testing (runs in Docker)
make test-server      # Run integration tests in Docker

# Production Docker operations
make build            # Build production Docker image
make push             # Push to GitHub Container Registry
make docker-all       # Build and push

# Kubernetes operations
make k8s-apply        # Deploy to Kubernetes cluster
make k8s-status       # Check deployment status
make k8s-logs         # Follow logs
make k8s-restart      # Rolling restart

# Local development (without Docker)
make install          # Install npm dependencies
make lint             # Run ESLint
make format           # Run Prettier

# Help
make help             # Show all available targets
```

### Manual Docker Compose

```bash
# Build and run
docker compose build
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

## API Endpoints

### Health Check
```
GET /health
```
Returns server status and last cache update time.

### Upcoming Events
```
GET /api/upcoming
```
Returns JSON array of upcoming events with details.

### Past Events
```
GET /api/past
```
Returns JSON array of past events with details.

### RSS Feeds
```
GET /feed/upcoming
GET /feed/past
```
Returns RSS feeds for upcoming and past events.

## Event Data Structure

Each event includes:
- `id`: Unique event identifier
- `title`: Event title
- `description`: Event description (HTML stripped)
- `dateTime`: Event start date/time (ISO 8601)
- `endTime`: Event end date/time (optional)
- `location`: Venue information (name, address, city, state)
- `eventUrl`: Direct link to event page
- `going`: Number of RSVPs
- `status`: "ACTIVE" or "PAST"

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

### Docker Development (Recommended)

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

### Local Development (without Docker)

For local development without Docker:

```bash
# Install dependencies
npm install

# Set MEETUP_ORG_URL environment variable
export MEETUP_ORG_URL="https://www.meetup.com/your-org-name"

# Run in development mode with hot reload
npm run dev

# Or build and run production
npm run build
npm start
```

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

**Requirements:** Docker and curl (no node/npm/jq needed on host)

### Code Quality

```bash
# Lint code
npm run lint

# Format code
npm run format
```

## How It Works

1. The server fetches the Meetup.com group page
2. Extracts the `__NEXT_DATA__` script tag containing GraphQL data
3. Parses the Apollo cache state to extract event information
4. Caches events in memory
5. Refreshes cache every hour (configurable)
6. Serves events via REST API and RSS feeds

## Architecture

- `src/index.ts`: Express server and route handlers
- `src/scraper.ts`: Meetup.com scraping logic
- `src/cache.ts`: Event caching mechanism
- `src/rss.ts`: RSS feed generation
- `src/types.ts`: TypeScript type definitions

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
