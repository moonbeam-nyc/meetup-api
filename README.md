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

### Prerequisites

- Docker and Docker Compose
- Make (optional, for easier workflow)

### Using Make (Recommended)

```bash
# Build, clean, and run (default workflow)
make dev

# View logs
make logs

# Stop the service
make stop

# Restart the service
make restart
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

Environment variables in `compose.yaml`:

- `MEETUP_ORG_URL`: Meetup.com group URL (required)
- `PORT`: Server port (default: 3000)
- `REFRESH_INTERVAL_HOURS`: Cache refresh interval (default: 1)

## Development

### Local Development (without Docker)

```bash
# Install dependencies
npm install

# Run in development mode with hot reload
npm run dev

# Build TypeScript
npm run build

# Run production build
npm start
```

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

## License

MIT
