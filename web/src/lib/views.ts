import type { Counts, ListQuery, ViewId } from '../types'

/*
 * A view is a query, not a mode.
 *
 * Keeping them one mechanism means navigation and filtering cannot disagree,
 * the URL keeps describing the whole screen, and picking a view leaves the
 * filter row usable rather than fighting it. queryForView and viewFromQuery are
 * inverses.
 */

const viewQuery: Record<ViewId, Partial<ListQuery>> = {
  all: {},
  not_started: { status: ['not_started'] },
  in_progress: { status: ['in_progress'] },
  completed: { status: ['completed'] },
  archived: { status: ['archived'] },
  overdue: { overdue: 'true' },
  blocked: { blocked: 'true' },
  recurring: { recurring: 'true' },
}

// The sort survives the move. The cursor does not: it is only valid for the
// query it was issued under.
export function queryForView(view: ViewId, from: ListQuery): ListQuery {
  return { ...viewQuery[view], sort: from.sort, dir: from.dir }
}

export function viewFromQuery(q: ListQuery): ViewId {
  if (q.overdue === 'true') return 'overdue'
  if (q.blocked === 'true') return 'blocked'
  if (q.recurring === 'true') return 'recurring'
  if (q.status?.length === 1) {
    const status = q.status[0]
    if (status === 'not_started' || status === 'in_progress') return status
    if (status === 'completed' || status === 'archived') return status
  }
  return 'all'
}

// View ids read like the status values they filter on; the counts payload is
// camelCase like the rest of the API. One mapping rather than bending either.
const countKeys: Record<ViewId, keyof Counts> = {
  all: 'all',
  not_started: 'notStarted',
  in_progress: 'inProgress',
  completed: 'completed',
  archived: 'archived',
  overdue: 'overdue',
  blocked: 'blocked',
  recurring: 'recurring',
}

export function countFor(counts: Counts | undefined, view: ViewId) {
  return counts?.[countKeys[view]]
}

const titles: Record<ViewId, string> = {
  all: 'All tasks',
  not_started: 'Not started',
  in_progress: 'In progress',
  completed: 'Completed',
  archived: 'Archived',
  overdue: 'Overdue',
  blocked: 'Blocked',
  recurring: 'Recurring',
}

export const viewTitle = (view: ViewId) => titles[view]
