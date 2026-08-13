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
  version: number
  createdAt: string
  updatedAt: string
}

// Writes carry no id, no version and no timestamps.
export type TodoInput = {
  name: string
  description: string
  dueDate: string | null
  status: Status
  priority: Priority
}
