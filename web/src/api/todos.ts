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

// A rejected write means the list is out of date too, but refetching while the
// panel is open would pull the ground out from under the banner explaining why.
// The caller refreshes when it is done with the answer.
export function useRefreshTodos() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: queryKeys.todos })
}

export const useCreateTodo = () =>
  useTodoMutation((input: TodoInput) =>
    request<Todo>('/todos', { method: 'POST', body: JSON.stringify(input) }),
  )

// Every write carries the version it was built from, so the server can reject
// it if the row moved on in the meantime.
export const useUpdateTodo = () =>
  useTodoMutation(({ id, version, input }: { id: number; version: number; input: TodoInput }) =>
    request<Todo>(`/todos/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ ...input, version }),
    }),
  )

export const useDeleteTodo = () =>
  useTodoMutation(({ id, version }: { id: number; version: number }) =>
    request<void>(`/todos/${id}?version=${version}`, { method: 'DELETE' }),
  )

// Its own route, not a status change, because completing a recurring task also
// opens the next one and the response says which.
export const useCompleteTodo = () =>
  useTodoMutation(({ id, version }: { id: number; version: number }) =>
    request<{ completed: Todo; spawned: Todo | null }>(`/todos/${id}/complete?version=${version}`, {
      method: 'POST',
    }),
  )
