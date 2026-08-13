import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { request } from './client'
import { MIN_SEARCH, queryKeys } from '../constants'
import type { DependencyView, ListQuery, Todo, TodoInput, TodoPage } from '../types'

// Turns the list state into a query string, dropping anything unset so the URL
// and the request both stay readable.
export function listParams(query: ListQuery, cursor?: string) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (Array.isArray(value)) value.forEach((v) => params.append(key, v))
    else if (value) params.set(key, value)
  }
  if (cursor) params.set('cursor', cursor)
  return params.toString()
}

/*
 * An infinite query from the start rather than a plain one made paged later.
 *
 * A keyset cursor is not an offset: the next page is fetched with a token from
 * the last row, so the swap replaces the list rather than appending to it, and
 * retrofitting that onto a useQuery means rewriting every caller.
 */
export function useTodos(query: ListQuery) {
  return useInfiniteQuery({
    queryKey: queryKeys.list(query),
    queryFn: ({ pageParam }) => request<TodoPage>(`/todos?${listParams(query, pageParam)}`),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor,
  })
}

function useTodoMutation<T>(fn: (v: T) => Promise<unknown>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.todos })
      // Deleting moves a task into the trash, so that list is stale too.
      qc.invalidateQueries({ queryKey: queryKeys.trash })
    },
  })
}

export function useTrash() {
  return useQuery({
    queryKey: queryKeys.trash,
    queryFn: () => request<{ items: Todo[] }>('/todos/trash').then((r) => r.items),
  })
}

export function useRestoreTodo() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => request<Todo>(`/todos/${id}/restore`, { method: 'POST' }),
    // Both lists move: one gains the task, the other loses it.
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.todos })
      qc.invalidateQueries({ queryKey: queryKeys.trash })
    },
  })
}

export function useDependencies(id: number | null) {
  return useQuery({
    queryKey: queryKeys.dependencies(id ?? 0),
    queryFn: () => request<DependencyView>(`/todos/${id}/dependencies`),
    enabled: id !== null,
  })
}

// The chain comes back from the write itself, so linking is one round trip
// rather than a mutation followed by a refetch. The task list still needs
// invalidating: the counter on the row has moved.
function useDependencyMutation<T>(fn: (v: T) => Promise<DependencyView>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (chain, vars) => {
      const { todoId } = vars as { todoId: number }
      qc.setQueryData(queryKeys.dependencies(todoId), chain)
      qc.invalidateQueries({ queryKey: queryKeys.todos })
    },
  })
}

export const useAddDependency = () =>
  useDependencyMutation(({ todoId, dependsOnId }: { todoId: number; dependsOnId: number }) =>
    request<DependencyView>(`/todos/${todoId}/dependencies`, {
      method: 'POST',
      body: JSON.stringify({ dependsOnId }),
    }),
  )

export const useRemoveDependency = () =>
  useDependencyMutation(({ todoId, dependsOnId }: { todoId: number; dependsOnId: number }) =>
    request<DependencyView>(`/todos/${todoId}/dependencies/${dependsOnId}`, { method: 'DELETE' }),
  )

// Held at the client until the term is long enough, so a one-letter search
// never reaches the database at all.
export function useTodoSearch(term: string, excludeId: number) {
  const trimmed = term.trim()
  return useQuery({
    queryKey: queryKeys.search(trimmed, excludeId),
    queryFn: () =>
      request<{ items: Todo[] }>(
        `/todos/search?q=${encodeURIComponent(trimmed)}&exclude=${excludeId}`,
      ).then((r) => r.items),
    enabled: trimmed.length >= MIN_SEARCH,
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
