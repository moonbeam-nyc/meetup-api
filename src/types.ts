export interface MeetupEvent {
  id: string;
  title: string;
  description: string;
  dateTime: string;
  endTime?: string;
  location?: {
    name?: string;
    address?: string;
    city?: string;
    state?: string;
  };
  eventUrl: string;
  going: number;
  status: 'ACTIVE' | 'PAST';
}

export interface EventsCache {
  upcoming: MeetupEvent[];
  past: MeetupEvent[];
  lastUpdated: Date;
}
