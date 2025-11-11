import axios from 'axios';
import * as cheerio from 'cheerio';
import { MeetupEvent } from './types';

export class MeetupScraper {
  private meetupUrl: string;

  constructor(meetupUrl: string) {
    this.meetupUrl = meetupUrl.endsWith('/') ? meetupUrl.slice(0, -1) : meetupUrl;
  }

  async scrapeEvents(): Promise<{ upcoming: MeetupEvent[]; past: MeetupEvent[] }> {
    try {
      console.log(`Fetching events from ${this.meetupUrl}...`);
      const response = await axios.get(this.meetupUrl, {
        headers: {
          'User-Agent':
            'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        },
      });

      const $ = cheerio.load(response.data);

      // Extract the __NEXT_DATA__ script tag that contains the GraphQL data
      const nextDataScript = $('script#__NEXT_DATA__').html();

      if (!nextDataScript) {
        throw new Error('Could not find __NEXT_DATA__ in the page');
      }

      const nextData = JSON.parse(nextDataScript);
      const apolloState = nextData?.props?.pageProps?.__APOLLO_STATE__;

      if (!apolloState) {
        throw new Error('Could not find Apollo state in the page data');
      }

      const upcoming = this.extractEventsFromApolloState(apolloState, 'ACTIVE');
      const past = this.extractEventsFromApolloState(apolloState, 'PAST');

      console.log(`Found ${upcoming.length} upcoming events and ${past.length} past events`);

      return { upcoming, past };
    } catch (error) {
      console.error('Error scraping events:', error);
      throw error;
    }
  }

  private extractEventsFromApolloState(
    apolloState: any,
    status: 'ACTIVE' | 'PAST'
  ): MeetupEvent[] {
    const events: MeetupEvent[] = [];

    // Find all Event objects in the Apollo cache
    for (const [key, value] of Object.entries(apolloState)) {
      if (key.startsWith('Event:') && typeof value === 'object' && value !== null) {
        const eventData = value as any;

        // Filter by status if provided
        if (eventData.status !== status) {
          continue;
        }

        // Extract venue information
        let location = undefined;
        if (eventData.venue && typeof eventData.venue === 'object') {
          const venueRef = eventData.venue.__ref;
          const venueData = venueRef ? apolloState[venueRef] : eventData.venue;

          if (venueData) {
            location = {
              name: venueData.name || undefined,
              address: venueData.address || undefined,
              city: venueData.city || undefined,
              state: venueData.state || undefined,
            };
          }
        }

        // Extract going count
        let going = 0;
        if (eventData.going) {
          going = typeof eventData.going === 'number' ? eventData.going : eventData.going.totalCount || 0;
        }

        events.push({
          id: eventData.id || key.replace('Event:', ''),
          title: eventData.title || 'Untitled Event',
          description: this.stripHtml(eventData.description || ''),
          dateTime: eventData.dateTime || eventData.startDate || '',
          endTime: eventData.endTime || eventData.endDate || undefined,
          location,
          eventUrl: eventData.eventUrl || `${this.meetupUrl}/events/${eventData.id}`,
          going,
          status: eventData.status || status,
        });
      }
    }

    // Sort by date (upcoming ascending, past descending)
    events.sort((a, b) => {
      const dateA = new Date(a.dateTime).getTime();
      const dateB = new Date(b.dateTime).getTime();
      return status === 'ACTIVE' ? dateA - dateB : dateB - dateA;
    });

    return events;
  }

  private stripHtml(html: string): string {
    const $ = cheerio.load(html);
    return $.text().trim();
  }
}
