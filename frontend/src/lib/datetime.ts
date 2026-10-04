// Date/clock helpers for API values. The backend sends calendar dates as
// `YYYY-MM-DD` and wall-clock times as `HH:MM`; neither carries a time zone, so
// they are handled as plain strings (clock values) or pinned to UTC (dates) to
// avoid the off-by-one-day shifts that local-time `Date` parsing causes.

const CLOCK_RE = /^(\d{2}):(\d{2})(?::\d{2})?$/;
const DATE_RE = /^(\d{4})-(\d{2})-(\d{2})$/;

/** Normalises `HH:MM` or `HH:MM:SS` to `HH:MM`. Other input is returned unchanged. */
export function formatClock(hhmm: string): string {
  const match = CLOCK_RE.exec(hhmm);
  if (!match) return hhmm;
  return `${match[1]}:${match[2]}`;
}

/** Formats a `YYYY-MM-DD` calendar date for display. Other input is returned unchanged. */
export function formatDate(isoDate: string, locale: string = 'id-ID'): string {
  const match = DATE_RE.exec(isoDate);
  if (!match) return isoDate;
  const [, y, m, d] = match;
  const utc = new Date(Date.UTC(Number(y), Number(m) - 1, Number(d)));
  if (Number.isNaN(utc.getTime())) return isoDate;
  return new Intl.DateTimeFormat(locale, {
    timeZone: 'UTC',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(utc);
}

/** Today's (or the given instant's) calendar date in the viewer's local time, as `YYYY-MM-DD`. */
export function localISODate(at: Date = new Date()): string {
  const y = at.getFullYear();
  const m = String(at.getMonth() + 1).padStart(2, '0');
  const d = String(at.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}
