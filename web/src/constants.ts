/*
 * The four schedules the brief names.
 *
 * Custom is the same mechanism with an interval above one and needs no separate
 * storage, but leaving it unnamed in the control makes a reader conclude it is
 * missing.
 */
export const SCHEDULES = [
  { key: 'none', label: 'Does not repeat', unit: null, interval: null },
  { key: 'day', label: 'Daily', unit: 'day', interval: 1 },
  { key: 'week', label: 'Weekly', unit: 'week', interval: 1 },
  { key: 'month', label: 'Monthly', unit: 'month', interval: 1 },
  { key: 'custom', label: 'Custom', unit: 'week', interval: 2 },
] as const

// Held back until three characters, which keeps the shortest and least
// selective searches from ever reaching the trigram index.
export const MIN_SEARCH = 3

// Long enough that typing does not recount the table, short enough that the
// rail is not visibly wrong.
export const COUNTS_STALE_TIME = 30_000

/*
 * Keys are hierarchical on purpose.
 *
 * Invalidation in TanStack Query matches by prefix, so invalidating ['todos']
 * reaches every list, chain and history underneath it. A write therefore needs
 * one invalidation rather than one per thing it could have changed, and a new
 * per-task query is covered the moment it is added.
 */
export const queryKeys = {
  todos: ['todos'],
  list: (query: object) => ['todos', 'list', query],
  counts: ['counts'],
  trash: ['trash'],
  dependencies: (id: number) => ['todos', id, 'dependencies'],
  events: (id: number) => ['todos', id, 'events'],
  search: (term: string, excludeId: number) => ['todos', 'search', term, excludeId],
}
