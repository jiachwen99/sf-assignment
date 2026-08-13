import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { request } from './client'
import { queryKeys } from '../constants'
import type { Todo, TodoInput } from '../types'

export function useTodos() {
  return useQuery({
    queryKey: queryKeys.todos,
    queryFn: () => request<{ items: Todo[] }>('/todos').then((r) => r.items),
  })
}

function useTodoMutation<T>(fn: (v: T) => Promise<unknown>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.todos }),
  })
}

export const useCreateTodo = () =>
  useTodoMutation((input: TodoInput) =>
    request<Todo>('/todos', { method: 'POST', body: JSON.stringify(input) }),
  )

export const useUpdateTodo = () =>
  useTodoMutation(({ id, input }: { id: number; input: TodoInput }) =>
    request<Todo>(`/todos/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  )

export const useDeleteTodo = () =>
  useTodoMutation((id: number) => request<void>(`/todos/${id}`, { method: 'DELETE' }))
