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
  sort?: SortField
  dir?: SortDir
}

export type TodoPage = {
  items: Todo[]
  nextCursor?: string
}
