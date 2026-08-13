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
  version: number
  createdAt: string
  updatedAt: string
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
