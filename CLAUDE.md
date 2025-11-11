# Meetup API Server - Project Knowledge Base

## Overview
This is a Node.js/TypeScript API server that scrapes Meetup.com organization pages and provides both JSON API endpoints and RSS feeds for upcoming and past events. It's designed to run in a Kubernetes cluster and be deployed to the moonbeam-nyc organization.

## Tech Stack
- **Runtime**: Node.js 20 (Alpine Linux)
- **Language**: TypeScript 5.3
- **Framework**: Express.js 4.18
- **Scraping**: Axios + Cheerio
- **Scheduling**: node-cron
- **RSS Generation**: rss package
- **Build**: TypeScript Compiler (tsc)
- **Container**: Docker (multi-stage build)
- **Orchestration**: Kubernetes
- **Registry**: GitHub Container Registry (ghcr.io/moonbeam-nyc)

## Project Structure

```
.
├── src/
│   ├── index.ts          # Main Express server and routes
│   ├── scraper.ts        # Meetup.com web scraper (extracts from __NEXT_DATA__)
│   ├── cache.ts          # Event caching mechanism
│   ├── rss.ts            # RSS feed generator
│   └── types.ts          # TypeScript type definitions
├── k8s/                  # Kubernetes manifests
│   ├── namespace.yaml    # meetup-api namespace
│   ├── deployment.yaml   # Deployment + ConfigMap
│   ├── service.yaml      # ClusterIP service
│   └── ingress.yaml      # Ingress configuration
├── Dockerfile            # Multi-stage Docker build
├── Makefile              # Build, deploy, and management targets
├── package.json          # Dependencies and scripts
└── tsconfig.json         # TypeScript configuration
```

## Core Functionality

### Web Scraping (src/scraper.ts)
- Scrapes Meetup.com organization pages by fetching the HTML and extracting the `__NEXT_DATA__` JSON embedded in the page
- Parses the Apollo GraphQL state from the Next.js data
- Extracts both ACTIVE (upcoming) and PAST events
- Parses event details: title, description, date/time, location, attendee count, event URL
- Strips HTML from descriptions for clean text output

### Caching (src/cache.ts)
- Maintains in-memory cache of upcoming and past events
- Automatically refreshes based on configurable interval (default: hourly)
- Provides getUpcoming(), getPast(), and getLastUpdated() methods

### API Endpoints (src/index.ts)
- `GET /` - API documentation
- `GET /health` - Health check with last updated timestamp
- `GET /api/upcoming` - JSON array of upcoming events
- `GET /api/past` - JSON array of past events
- `GET /feed/upcoming` - RSS feed of upcoming events
- `GET /feed/past` - RSS feed of past events

### RSS Generation (src/rss.ts)
- Converts event data to RSS 2.0 format
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
1. **Builder stage**: Installs all dependencies and builds TypeScript
2. **Production stage**: Only includes production dependencies and compiled JS

Features:
- Non-root user (nodejs:1001)
- Minimal attack surface
- Health check integration
- Optimized for size and security

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
- `make install` - Install npm dependencies (local)
- `make lint` - Run ESLint (local)
- `make format` - Run Prettier (local)
- `make test` - Run unit tests (npm test)
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
1. Install dependencies: `npm install`
2. Set `MEETUP_ORG_URL` environment variable: `export MEETUP_ORG_URL="https://www.meetup.com/your-org"`
3. Build TypeScript: `npm run build`
4. Run server: `npm start`

Note: Docker development is the primary workflow. Local development is supported but not the recommended approach.

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
