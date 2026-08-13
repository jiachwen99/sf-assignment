const DAY = 86_400_000

const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()

const daysBetween = (from: Date, to: Date) => Math.round((startOfDay(to) - startOfDay(from)) / DAY)

const shortDate = (d: Date, now: Date) =>
  d.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    ...(d.getFullYear() === now.getFullYear() ? {} : { year: 'numeric' }),
  })

/*
 * Relative dates, not absolute ones.
 *
 * The list sorts by due date, so any screen of it shares roughly one date and
 * an absolute column reads as though every row is identical. "In 3 days" and
 * "5 days late" answer the question the user actually has, and they differ row
 * to row so the column carries information again.
 */
export type DueState = 'none' | 'late' | 'today' | 'soon' | 'later'

export function describeDue(iso: string | null, now = new Date()): { text: string; state: DueState } {
  if (!iso) return { text: 'No date', state: 'none' }

  const due = new Date(iso)
  const days = daysBetween(now, due)

  // Relative only while it is still actionable. Past about a fortnight the
  // relative form stops discriminating: a screen of "200 days late" is exactly
  // as useless as a screen of identical timestamps, which is what this
  // replaced. The date is the more informative answer at that range.
  if (days < -14) return { text: shortDate(due, now), state: 'late' }
  if (days < 0) {
    const late = Math.abs(days)
    return { text: late === 1 ? '1 day late' : `${late} days late`, state: 'late' }
  }
  if (days === 0) return { text: 'Today', state: 'today' }
  if (days === 1) return { text: 'Tomorrow', state: 'soon' }
  if (days <= 14) return { text: `In ${days} days`, state: 'soon' }
  return { text: shortDate(due, now), state: 'later' }
}

// Created only ever runs backwards, so it says how long ago rather than
// borrowing the due-date vocabulary, which has to handle both directions.
export function describeCreated(iso: string, now = new Date()) {
  const created = new Date(iso)
  const days = -daysBetween(now, created)

  if (days <= 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days <= 14) return `${days} days ago`
  return shortDate(created, now)
}

// The absolute value still matters, so it lives in the title attribute rather
// than in the column.
export function exactDue(iso: string | null) {
  if (!iso) return 'No due date'
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'full', timeStyle: 'short' })
}

// datetime-local wants "YYYY-MM-DDTHH:mm" in local time, not an ISO instant.
export function toLocalInput(iso: string | null) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
