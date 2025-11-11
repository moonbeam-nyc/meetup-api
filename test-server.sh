#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
PORT=3000
MAX_WAIT=10
CHECK_INTERVAL=1

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}Stopping docker compose...${NC}"
    docker compose down
}

# Set trap to cleanup on exit
trap cleanup EXIT INT TERM

echo -e "${GREEN}Starting server test with Docker Compose...${NC}"
echo ""

# Start docker compose in detached mode
echo -e "${YELLOW}Starting docker compose...${NC}"
docker compose up -d --build

# Wait for server to be ready
echo -e "${YELLOW}Waiting for server to be ready (max ${MAX_WAIT}s)...${NC}"
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
    if curl -s -f http://localhost:$PORT/health > /dev/null 2>&1; then
        echo -e "${GREEN}Server is ready!${NC}"
        break
    fi
    sleep $CHECK_INTERVAL
    ELAPSED=$((ELAPSED + CHECK_INTERVAL))
    printf "."
done
echo ""

if [ $ELAPSED -ge $MAX_WAIT ]; then
    echo -e "${RED}Server failed to start within $MAX_WAIT seconds${NC}"
    echo "Docker compose logs:"
    docker compose logs
    exit 1
fi

echo ""

# Function to test an endpoint
test_endpoint() {
    local endpoint=$1
    local description=$2
    local check_type=${3:-"any"}  # any, json, xml

    echo -e "${YELLOW}Testing $description...${NC}"

    # Get HTTP status and response
    http_code=$(curl -s -o /tmp/test-response.txt -w "%{http_code}" http://localhost:$PORT$endpoint)

    if [ "$http_code" != "200" ]; then
        echo -e "${RED}  ✗ Failed: HTTP $http_code${NC}"
        return 1
    fi

    response=$(cat /tmp/test-response.txt)

    if [ -z "$response" ]; then
        echo -e "${RED}  ✗ Failed: Empty response${NC}"
        return 1
    fi

    # Type-specific checks
    if [ "$check_type" = "json" ]; then
        if echo "$response" | grep -q "^{"; then
            echo -e "${GREEN}  ✓ Success: Valid JSON response (HTTP 200)${NC}"
        else
            echo -e "${RED}  ✗ Failed: Not valid JSON${NC}"
            return 1
        fi
    elif [ "$check_type" = "xml" ]; then
        if echo "$response" | grep -q "<rss"; then
            echo -e "${GREEN}  ✓ Success: Valid RSS feed (HTTP 200)${NC}"
        else
            echo -e "${RED}  ✗ Failed: Not valid RSS XML${NC}"
            return 1
        fi
    else
        echo -e "${GREEN}  ✓ Success: Response received (HTTP 200)${NC}"
    fi

    return 0
}

# Run tests
echo -e "${GREEN}Running endpoint tests...${NC}"
echo ""

FAILED=0

# Test health endpoint
test_endpoint "/health" "Health check" "json" || FAILED=$((FAILED + 1))

# Test root endpoint
test_endpoint "/" "API documentation" "json" || FAILED=$((FAILED + 1))

# Test upcoming events API
test_endpoint "/api/upcoming" "Upcoming events API" "json" || FAILED=$((FAILED + 1))

# Test past events API
test_endpoint "/api/past" "Past events API" "json" || FAILED=$((FAILED + 1))

# Test upcoming events RSS feed
test_endpoint "/feed/upcoming" "Upcoming events RSS feed" "xml" || FAILED=$((FAILED + 1))

# Test past events RSS feed
test_endpoint "/feed/past" "Past events RSS feed" "xml" || FAILED=$((FAILED + 1))

echo ""
echo "=========================================="

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo -e "${RED}$FAILED test(s) failed ✗${NC}"
    exit 1
fi
