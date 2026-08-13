import type { Priority, Status } from '../types'

// The words the interface uses for the two enumerations, in one place, so a
// select and a table cell can never disagree about what a value is called.

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
