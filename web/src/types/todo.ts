export type Status = 'not_started' | 'in_progress' | 'completed' | 'archived'
export type Priority = 'low' | 'medium' | 'high'
export type RecurUnit = 'day' | 'week' | 'month'

export type Todo = {
  id: number
  name: string
  description: string
  dueDate: string | null
  status: Status
  priority: Priority
  recurUnit: RecurUnit | null
  recurInterval: number | null
  unmetDeps: number
  version: number
  createdAt: string
  updatedAt: string
}

// A dependency or dependent, named. Enough to draw a chain node and to say what
// is holding a task up, without loading whole tasks to do it.
export type Blocker = {
  id: number
  name: string
  status: Status
  // A deleted task still blocks what waits on it, so the chain has to say so.
  deleted: boolean
}

export type DependencyView = {
  dependencies: Blocker[]
  dependents: Blocker[]
}

// The editable fields only. The version travels beside this rather than in it,
// because it says which copy you edited, not what the task is.
export type TodoInput = {
  name: string
  description: string
  dueDate: string | null
  status: Status
  priority: Priority
  recurUnit: RecurUnit | null
  recurInterval: number | null
}
