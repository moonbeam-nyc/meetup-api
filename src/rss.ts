import RSS from 'rss';
import { MeetupEvent } from './types';

export function generateRSSFeed(
  events: MeetupEvent[],
  meetupUrl: string,
  feedType: 'upcoming' | 'past'
): string {
  const groupName = extractGroupName(meetupUrl);

  const feed = new RSS({
    title: `${groupName} - ${feedType === 'upcoming' ? 'Upcoming' : 'Past'} Events`,
    description: `${feedType === 'upcoming' ? 'Upcoming' : 'Past'} meetup events for ${groupName}`,
    feed_url: `${meetupUrl}/feed/${feedType}`,
    site_url: meetupUrl,
    language: 'en',
    pubDate: new Date().toISOString(),
  });

  events.forEach((event) => {
    const locationStr = event.location
      ? `${event.location.name || ''} ${event.location.address || ''} ${event.location.city || ''}, ${event.location.state || ''}`.trim()
      : 'Location TBD';

    feed.item({
      title: event.title,
      description: `${event.description}\n\nWhen: ${new Date(event.dateTime).toLocaleString()}\nWhere: ${locationStr}\nRSVPs: ${event.going}`,
      url: event.eventUrl,
      date: event.dateTime,
      guid: event.id,
    });
  });

  return feed.xml();
}

function extractGroupName(url: string): string {
  const match = url.match(/meetup\.com\/([^/]+)/);
  return match ? match[1] : 'Meetup Group';
}
