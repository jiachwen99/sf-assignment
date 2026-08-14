import type { Todo } from './todo'

export type SortField = 'created' | 'due' | 'priority' | 'status' | 'name'
export type SortDir = 'asc' | 'desc'

// Everything that describes which rows are on screen, which is also everything
// that belongs in the URL.
export type ListQuery = {
  name?: string
  status?: string[]
  priority?: string[]
  dueFrom?: string
  dueTo?: string
  blocked?: string
  recurring?: string
  overdue?: string
  sort?: SortField
  dir?: SortDir
}

export type TodoPage = {
  items: Todo[]
  nextCursor?: string
}

export type Counts = {
  all: number
  notStarted: number
  inProgress: number
  completed: number
  archived: number
  overdue: number
  blocked: number
  recurring: number
  trash: number
}

export type ViewId =
  | 'all'
  | 'not_started'
  | 'in_progress'
  | 'completed'
  | 'archived'
  | 'overdue'
  | 'blocked'
  | 'recurring'

export type BulkResult = {
  id: number
  ok: boolean
  error?: string
  blockers?: { id: number; name: string }[]
}
