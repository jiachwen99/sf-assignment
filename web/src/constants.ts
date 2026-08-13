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

export const queryKeys = {
  // Every list state is its own cache entry, so changing a filter or a sort
  // starts a fresh page rather than appending to somebody else's.
  todos: ['todos'],
  list: (query: object) => ['todos', 'list', query],
  counts: ['counts'],
  trash: ['trash'],
  dependencies: (id: number) => ['todos', id, 'dependencies'],
  search: (term: string, excludeId: number) => ['todos', 'search', term, excludeId],
}
