import { formatClock, formatDate, localISODate } from '../datetime';

describe('formatClock', () => {
  it('drops seconds from HH:MM:SS', () => {
    expect(formatClock('09:00:00')).toBe('09:00');
  });

  it('keeps HH:MM unchanged', () => {
    expect(formatClock('17:30')).toBe('17:30');
  });

  it('returns unrecognised input unchanged', () => {
    expect(formatClock('')).toBe('');
    expect(formatClock('9am')).toBe('9am');
  });
});

describe('formatDate', () => {
  it('formats the calendar day without a local-time shift', () => {
    // Parsed as a Date, `2024-01-01` is UTC midnight, which a viewer west of
    // UTC sees as 31 Dec 2023. formatDate must still show 1 Jan 2024.
    const shifted = new Intl.DateTimeFormat('en-US', { timeZone: 'America/Los_Angeles' }).format(new Date('2024-01-01'));
    expect(shifted).toBe('12/31/2023');

    expect(formatDate('2024-01-01', 'en-US')).toBe('Jan 1, 2024');
  });

  it('formats with the default Indonesian locale', () => {
    expect(formatDate('2025-08-17')).toMatch(/17/);
    expect(formatDate('2025-08-17')).toMatch(/2025/);
  });

  it('returns unrecognised input unchanged', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
  });
});

describe('localISODate', () => {
  it('uses local calendar fields', () => {
    expect(localISODate(new Date(2024, 1, 29, 23, 30))).toBe('2024-02-29');
  });
});
