# Meetup API Server - Project Knowledge Base

## Overview
This is a lightweight Go API server that scrapes Meetup.com organization pages and provides both JSON API endpoints and RSS feeds for upcoming and past events.

## Deployment (live)
**Primary/live deployment is the Pi** (fleet at `~/projects/personal`), NOT
Kubernetes. It runs as a native systemd service — the old Cloudflare tunnel to a
Hetzner VPS was retired in favour of hosting on the Pi.

- **Public URL:** `https://meetup-api.astoria.app`, fronted by the `http-routing`
  Caddy + cloudflared tunnel (orange-cloud CNAME → the Pi's tunnel). Caddy
  reverse-proxies the host → `127.0.0.1:3000`.
- **Service:** `systemd/meetup-api.service` runs the Go binary directly (no
  Docker). Build + install (one-time, sudo):
  ```bash
  go build -ldflags="-w -s" -o meetup-api ./cmd/server
  sudo cp systemd/meetup-api.service /etc/systemd/system/
  sudo systemctl enable --now meetup-api.service
  ```
  After a code change: rebuild the binary, then `sudo systemctl restart meetup-api`.
- **Env** (set in the unit): `PORT=3000`,
  `MEETUP_ORG_URL=https://www.meetup.com/astoria-tech-meetup/`,
  `REFRESH_INTERVAL_HOURS=1`.
- **Startup note:** `main.go` does the full initial scrape *before* it calls
  `ListenAndServe`, so the port isn't accepting connections for ~30s after start
  (paginates all past events). Normal — not a hang.
- The Docker/`compose.yaml` and `k8s/` manifests below are kept for reference /
  the moonbeam-nyc org path, but are **not** how the Pi deployment runs.

## Tech Stack
- **Runtime**: Go 1.21
- **Language**: Go (Golang)
- **HTTP Server**: Go standard library (net/http)
- **Scraping**: goquery (jQuery-like HTML parsing)
- **Scheduling**: robfig/cron
- **RSS Generation**: gorilla/feeds
- **Build**: Go compiler with static linking
- **Container**: Docker (multi-stage build with scratch base)
- **Image Size**: 6MB (26x smaller than Node.js version)
- **Orchestration**: Kubernetes
- **Registry**: GitHub Container Registry (ghcr.io/moonbeam-nyc)

## Project Structure

```
.
├── cmd/
│   └── server/
│       ├── main.go       # HTTP server, routes, and cache management
│       ├── scraper.go    # Meetup.com web scraper (extracts from __NEXT_DATA__)
│       ├── rss.go        # RSS feed generator
│       └── types.go      # Go type definitions
├── k8s/                  # Kubernetes manifests
│   ├── namespace.yaml    # meetup-api namespace
│   ├── deployment.yaml   # Deployment + ConfigMap
│   ├── service.yaml      # ClusterIP service
│   ├── ingress.yaml      # Ingress configuration
│   └── custom-headers.yaml # Custom HTTP headers ConfigMap
├── Dockerfile            # Multi-stage Docker build (scratch-based)
├── Makefile              # Build, deploy, and management targets
├── go.mod                # Go module dependencies
└── go.sum                # Go module checksums
```

## Core Functionality

### Web Scraping (cmd/server/scraper.go)
- **Upcoming (ACTIVE)** events: fetch the org page HTML, extract the `__NEXT_DATA__`
  JSON, parse the Apollo state (goquery), filter by status. Robust — no private API.
- **Past events (full history):** POST to Meetup's `gql2` GraphQL endpoint with
  cursor pagination (`getPastGroupEvents`).
- Parses event details: title, description, date/time, location, attendee count,
  event URL; strips HTML from descriptions.

#### ⚠️ Past-events GraphQL uses Automatic Persisted Queries (APQ) — and the fallback
The GraphQL call sends only the query's **sha256 hash** (`84d621…`), not the query
text. When Meetup's edge cache doesn't have that hash registered, it returns
**HTTP 200 with `errors: [PersistedQueryNotFound]`** and zero edges. This is
intermittent — the hash is warm only when real browsers recently ran it. Two guards
handle this (both added 2026-07-05, do not remove without a replacement):
1. `fetchPastEventsGraphQL` inspects the GraphQL `errors` array and returns a real
   error on `PersistedQueryNotFound` (previously the empty result was silently
   cached as "no past events" — the bug that shipped 0 past events on first deploy).
2. `fetchAllPastEvents` falls back to scraping the **`/events/?type=past` HTML
   page** (its `__NEXT_DATA__` embeds the ~10 most recent past events, no APQ), so a
   cold cache degrades to "recent events", never zero. Full 234-event history
   returns once the hash is warm again.
- A truly robust fix would implement APQ properly (resend with the full query text
  on `PersistedQueryNotFound`), but that needs Meetup's exact query document and is
  fragile to their changes — the fallback is the pragmatic choice.

### Caching (cmd/server/main.go)
- Maintains in-memory cache of upcoming and past events with RWMutex for thread safety
- Automatically refreshes based on configurable interval (default: hourly)
- **Defensive `refreshCache` (never serve/keep empty):**
  - A scrape error leaves the existing cache untouched.
  - Empty `upcoming` never overwrites a populated cache (upcoming *can* legitimately
    be 0, but we treat an empty scrape as a transient failure).
  - `past` is **monotonic**: only accept a past list ≥ what's cached, so the ~10-item
    HTML fallback can't clobber a good full history, yet still upgrades once GraphQL
    recovers. (Past history only accumulates, so a shrink signals a degraded fetch.)
  - **Initial load retries** (5× with backoff) instead of `log.Fatal` — a cold APQ
    cache at startup no longer takes the server down or serves 0 past events.

### API Endpoints (cmd/server/main.go)
- `GET /` - API documentation
- `GET /health` - Health check with last updated timestamp
- `GET /api/upcoming` - JSON array of upcoming events
- `GET /api/past` - JSON array of past events
- `GET /feed/upcoming` - RSS feed of upcoming events
- `GET /feed/past` - RSS feed of past events
- All responses include `Cache-Control` headers for CDN caching

### RSS Generation (cmd/server/rss.go)
- Converts event data to RSS 2.0 format using gorilla/feeds
- Includes event title, description, date, location, and link

## Environment Variables

### Required
- `MEETUP_ORG_URL` - The Meetup.com organization URL to scrape (e.g., "https://www.meetup.com/astoria-tech-meetup")

### Optional
- `PORT` - Server port (default: 3000)
- `REFRESH_INTERVAL_HOURS` - How often to refresh the cache (default: 1 hour)

## Build & Deployment

### Docker Build
The Dockerfile uses a multi-stage build:
1. **Builder stage**: Uses golang:1.21-alpine to compile Go code
2. **Production stage**: Uses scratch (empty base image) with only the binary

Features:
- Single static binary (no dependencies)
- Non-root user (nobody:65534)
- Final image size: **6MB** (vs 159MB Node.js version)
- No shell or package manager (maximum security)
- CA certificates included for HTTPS

### Makefile Targets

**Docker Operations:**
- `make build` - Build Docker image with version from package.json
- `make tag` - Tag image as latest
- `make push` - Push to GitHub Container Registry
- `make docker-all` - Build and push in one command

**Kubernetes Operations:**
- `make k8s-apply` - Apply all manifests (creates namespace, deployment, service, ingress)
- `make k8s-delete` - Delete all resources
- `make k8s-restart` - Rolling restart of deployment
- `make k8s-logs` - Follow logs from deployment
- `make k8s-status` - Show all resources in namespace

**Development:**
- `make dev` - Run dev server in Docker Compose
- `make dev-build` - Build and run dev server in Docker
- `make dev-logs` - View Docker Compose logs
- `make dev-down` - Stop Docker Compose services
- `make go-run` - Run Go server locally (requires Go)
- `make go-build` - Build Go binary locally
- `make go-fmt` - Format Go code
- `make test-server` - Integration test in Docker (starts compose, validates endpoints, cleans up)

### Kubernetes Architecture

**Namespace:** `meetup-api`
- Isolated namespace for all resources

**Deployment:**
- 2 replicas for high availability
- Rolling update strategy (maxSurge: 1, maxUnavailable: 0)
- Resource limits: 256Mi RAM, 500m CPU
- Resource requests: 128Mi RAM, 100m CPU
- Liveness probe on `/health` endpoint
- Readiness probe on `/health` endpoint
- Runs as non-root user (1001)
- Security context with dropped capabilities

**ConfigMap:**
- Stores MEETUP_ORG_URL and REFRESH_INTERVAL_HOURS
- Update ConfigMap and restart deployment to change configuration

**Service:**
- ClusterIP type (internal only)
- Port 80 → Container port 3000

**Ingress:**
- Configurable hostname (update `meetup-api.example.com`)
- Nginx ingress class by default
- TLS configuration commented out (uncomment for HTTPS)

## Development Workflow

### Local Development with Docker (Recommended)
1. Configure `compose.yaml` with your `MEETUP_ORG_URL`
2. Run dev server: `make dev`
3. Test endpoints at http://localhost:3000
4. View logs: `make dev-logs`
5. Stop server: `make dev-down`

### Local Development without Docker (Optional)
1. Install Go 1.21+ from https://golang.org/dl/
2. Set `MEETUP_ORG_URL` environment variable: `export MEETUP_ORG_URL="https://www.meetup.com/your-org"`
3. Run server: `make go-run`
4. Or build binary: `make go-build && ./meetup-api`

Note: Docker development is the primary workflow. Local Go development is supported but not required.

### Testing
The project includes an integration test script (`test-server.sh`) that runs entirely in Docker:

```bash
make test-server
```

The test script:
1. Starts the server in Docker Compose (detached mode with `--build`)
2. Waits for the `/health` endpoint to respond (max 10 seconds)
3. Tests all API endpoints (`/`, `/health`, `/api/upcoming`, `/api/past`)
4. Tests all RSS feed endpoints (`/feed/upcoming`, `/feed/past`)
5. Validates HTTP 200 responses
6. Checks JSON responses start with `{`
7. Checks RSS feeds contain `<rss` tag
8. Stops Docker Compose and cleans up

**Requirements:**
- Docker and Docker Compose
- curl (standard on macOS/Linux)
- No node/npm/jq required on host machine

**Configuration:**
The test uses `compose.yaml` which already has `MEETUP_ORG_URL` configured (defaults to astoria-tech-meetup).

### Building for Production
1. Update version in package.json
2. Build: `make build`
3. Push to registry: `make push`

### Deploying to Kubernetes
1. Ensure kubectl is configured for your cluster
2. Update ConfigMap in `k8s/deployment.yaml` with correct MEETUP_ORG_URL
3. Update Ingress in `k8s/ingress.yaml` with your domain
4. Deploy: `make k8s-apply`
5. Check status: `make k8s-status`
6. View logs: `make k8s-logs`

### Updating the Deployment
1. Make code changes
2. Update version in package.json
3. Build and push: `make docker-all`
4. Update image tag in `k8s/deployment.yaml` if using specific version
5. Apply changes: `make k8s-apply`
6. Or just restart to pull latest: `make k8s-restart`

## Security Considerations

### Docker
- Multi-stage build minimizes image size
- Only production dependencies in final image
- Non-root user
- Health checks for reliability
- No unnecessary packages

### Kubernetes
- Runs as non-root (UID 1001)
- Read-only root filesystem disabled (app needs to write logs)
- All capabilities dropped
- No privilege escalation
- Resource limits prevent resource exhaustion

### Environment Variables
- Primary configuration is in `compose.yaml` for Docker development
- Sensitive data should be in Kubernetes Secrets, not ConfigMaps
- .envrc, .env, and .env.local are gitignored (for optional local development)

## Monitoring & Debugging

### Health Checks
- `/health` endpoint returns status and last cache update time
- Used by Docker HEALTHCHECK and Kubernetes probes

### Logs
- All events logged to stdout
- Kubernetes logs: `make k8s-logs`
- Docker logs: `docker logs <container>`

### Common Issues
1. **"MEETUP_ORG_URL environment variable is required"** - ConfigMap not set or deployment not configured
2. **Scraping fails** - Meetup.com may have changed their page structure, check __NEXT_DATA__ format
3. **Cache not updating** - Check cron schedule syntax and logs for errors
4. **Only ~10 past events (not the full history)** - Meetup's persisted-query cache
   is cold (`PersistedQueryNotFound` in logs); the HTML fallback is serving recent
   events. Expected to self-recover on a later refresh once the hash warms. See the
   APQ note under Web Scraping.

## Git Configuration
- Do NOT include Claude Code footer in commit messages
- Wait for verification before committing fixes
- Use "test, add, commit" workflow: ensure tests pass before committing

## Future Enhancements
- [ ] Add tests (jest/mocha)
- [ ] Add Prometheus metrics endpoint
- [ ] Add rate limiting
- [ ] Add database for persistent storage
- [ ] Add authentication for write operations
- [ ] Add Helm chart for easier deployment
- [ ] Add GitHub Actions for CI/CD
- [ ] Add support for multiple Meetup organizations
