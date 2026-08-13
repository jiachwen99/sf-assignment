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

export const queryKeys = {
  todos: ['todos'],
  counts: ['counts'],
  trash: ['trash'],
}
