import { EventsCache, MeetupEvent } from './types';
import { MeetupScraper } from './scraper';

export class EventCache {
  private cache: EventsCache = {
    upcoming: [],
    past: [],
    lastUpdated: new Date(0),
  };

  private scraper: MeetupScraper;

  constructor(meetupUrl: string) {
    this.scraper = new MeetupScraper(meetupUrl);
  }

  async refresh(): Promise<void> {
    try {
      console.log('Refreshing event cache...');
      const { upcoming, past } = await this.scraper.scrapeEvents();

      this.cache = {
        upcoming,
        past,
        lastUpdated: new Date(),
      };

      console.log(`Cache refreshed at ${this.cache.lastUpdated.toISOString()}`);
    } catch (error) {
      console.error('Failed to refresh cache:', error);
      throw error;
    }
  }

  getUpcoming(): MeetupEvent[] {
    return this.cache.upcoming;
  }

  getPast(): MeetupEvent[] {
    return this.cache.past;
  }

  getLastUpdated(): Date {
    return this.cache.lastUpdated;
  }
}
