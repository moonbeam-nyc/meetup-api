import express, { Request, Response } from 'express';
import cron from 'node-cron';
import { EventCache } from './cache';
import { generateRSSFeed } from './rss';

const PORT = process.env.PORT || 3000;
const MEETUP_ORG_URL = process.env.MEETUP_ORG_URL;
const REFRESH_INTERVAL_HOURS = parseInt(process.env.REFRESH_INTERVAL_HOURS || '1', 10);

if (!MEETUP_ORG_URL) {
  console.error('MEETUP_ORG_URL environment variable is required');
  process.exit(1);
}

const app = express();
const eventCache = new EventCache(MEETUP_ORG_URL);

// Middleware
app.use(express.json());

// Add cache headers for all responses (1 minute cache for Cloudflare)
app.use((_req: Request, res: Response, next) => {
  res.set('Cache-Control', 'public, max-age=60, s-maxage=60');
  next();
});

// Root endpoint - API documentation
app.get('/', (_req: Request, res: Response) => {
  res.json({
    name: 'Meetup API Server',
    version: '1.0.0',
    endpoints: {
      health: '/health',
      api: {
        upcoming: '/api/upcoming',
        past: '/api/past',
      },
      rss: {
        upcoming: '/feed/upcoming',
        past: '/feed/past',
      },
    },
    meetupUrl: MEETUP_ORG_URL,
  });
});

// Health check endpoint
app.get('/health', (_req: Request, res: Response) => {
  res.json({
    status: 'ok',
    lastUpdated: eventCache.getLastUpdated().toISOString(),
    meetupUrl: MEETUP_ORG_URL,
  });
});

// API endpoint for upcoming events
app.get('/api/upcoming', (_req: Request, res: Response) => {
  const events = eventCache.getUpcoming();
  res.json({
    events,
    count: events.length,
    lastUpdated: eventCache.getLastUpdated().toISOString(),
  });
});

// API endpoint for past events
app.get('/api/past', (_req: Request, res: Response) => {
  const events = eventCache.getPast();
  res.json({
    events,
    count: events.length,
    lastUpdated: eventCache.getLastUpdated().toISOString(),
  });
});

// RSS feed for upcoming events
app.get('/feed/upcoming', (_req: Request, res: Response) => {
  const events = eventCache.getUpcoming();
  const rss = generateRSSFeed(events, MEETUP_ORG_URL, 'upcoming');
  res.type('application/rss+xml');
  res.send(rss);
});

// RSS feed for past events
app.get('/feed/past', (_req: Request, res: Response) => {
  const events = eventCache.getPast();
  const rss = generateRSSFeed(events, MEETUP_ORG_URL, 'past');
  res.type('application/rss+xml');
  res.send(rss);
});

// Initialize cache and start server
async function start() {
  try {
    console.log('Starting Meetup API Server...');
    console.log(`Meetup URL: ${MEETUP_ORG_URL}`);
    console.log(`Refresh interval: ${REFRESH_INTERVAL_HOURS} hour(s)`);

    // Initial cache load
    await eventCache.refresh();

    // Schedule hourly refresh
    cron.schedule(`0 */${REFRESH_INTERVAL_HOURS} * * *`, async () => {
      console.log('Running scheduled cache refresh...');
      try {
        await eventCache.refresh();
      } catch (error) {
        console.error('Scheduled refresh failed:', error);
      }
    });

    // Start Express server
    app.listen(PORT, () => {
      console.log(`Server running on http://localhost:${PORT}`);
      console.log(`API endpoints:`);
      console.log(`  - GET /health`);
      console.log(`  - GET /api/upcoming`);
      console.log(`  - GET /api/past`);
      console.log(`  - GET /feed/upcoming`);
      console.log(`  - GET /feed/past`);
    });
  } catch (error) {
    console.error('Failed to start server:', error);
    process.exit(1);
  }
}

start();
