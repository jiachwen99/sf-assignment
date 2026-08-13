import type { Priority, Status } from '../types'

const statusLabels: Record<Status, string> = {
  not_started: 'Not started',
  in_progress: 'In progress',
  completed: 'Completed',
  archived: 'Archived',
}

const priorityLabels: Record<Priority, string> = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
}

export const statusLabel = (s: Status) => statusLabels[s]
export const priorityLabel = (p: Priority) => priorityLabels[p]

export const statusOptions = Object.entries(statusLabels).map(([value, label]) => ({ value, label }))
export const priorityOptions = Object.entries(priorityLabels).map(([value, label]) => ({
  value,
  label,
}))

const DAY = 86_400_000

/*
 * Relative dates, not absolute ones.
 *
 * The list sorts by due date, so any screen of it shares roughly one date and
 * an absolute column reads as though every row is identical. "In 3 days" and
 * "5 days late" answer the question the user actually has, and they differ row
 * to row so the column carries information again.
 */
type DueState = 'none' | 'late' | 'today' | 'soon' | 'later'

export function describeDue(iso: string | null, now = new Date()): { text: string; state: DueState } {
  if (!iso) return { text: 'No date', state: 'none' }

  const due = new Date(iso)
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const startOfDue = new Date(due.getFullYear(), due.getMonth(), due.getDate()).getTime()
  const days = Math.round((startOfDue - startOfToday) / DAY)

  const asDate = () =>
    due.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      ...(due.getFullYear() === now.getFullYear() ? {} : { year: 'numeric' }),
    })

  // Relative only while it is still actionable. Past about a fortnight the
  // relative form stops discriminating: a screen of "200 days late" is exactly
  // as useless as a screen of identical timestamps, which is what this
  // replaced. The date is the more informative answer at that range.
  if (days < -14) return { text: asDate(), state: 'late' }
  if (days < 0) {
    const late = Math.abs(days)
    return { text: late === 1 ? '1 day late' : `${late} days late`, state: 'late' }
  }
  if (days === 0) return { text: 'Today', state: 'today' }
  if (days === 1) return { text: 'Tomorrow', state: 'soon' }
  if (days <= 14) return { text: `In ${days} days`, state: 'soon' }
  return { text: asDate(), state: 'later' }
}

// Created only ever runs backwards, so it says how long ago rather than
// borrowing the due-date vocabulary, which has to handle both directions.
export function describeCreated(iso: string, now = new Date()) {
  const created = new Date(iso)
  const days = Math.round(
    (new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime() -
      new Date(created.getFullYear(), created.getMonth(), created.getDate()).getTime()) /
      DAY,
  )

  if (days <= 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days <= 14) return `${days} days ago`
  return created.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    ...(created.getFullYear() === now.getFullYear() ? {} : { year: 'numeric' }),
  })
}

// The absolute value still matters, so it lives in the title attribute rather
// than in the column.
export function exactDue(iso: string | null) {
  if (!iso) return 'No due date'
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'full', timeStyle: 'short' })
}
