export type Status = 'not_started' | 'in_progress' | 'completed' | 'archived'
export type Priority = 'low' | 'medium' | 'high'

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

// The editable fields only. The version travels beside this rather than in it,
// because it says which copy you edited, not what the task is.
export type TodoInput = {
  name: string
  description: string
  dueDate: string | null
  status: Status
  priority: Priority
}
