import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { queryKeys } from '../constants'
import type { User } from '../types'
import { request } from './client'

/*
 * Signing in is optional. The API answers null rather than an error when nobody
 * is, because not being signed in is a state the application supports: accounts
 * supply identity, never separation, and every account sees the same tasks.
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: queryKeys.currentUser,
    queryFn: () => request<User | null>('/auth/me'),
  })
}

// Signing in or out changes who the history names, so it invalidates everything
// rather than only the account.
function useSessionMutation<T>(fn: (v: T) => Promise<unknown>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.currentUser })
      qc.invalidateQueries({ queryKey: queryKeys.todos })
    },
  })
}

export const useLogin = () =>
  useSessionMutation((body: { email: string; password: string }) =>
    request<User>('/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  )

export const useRegister = () =>
  useSessionMutation((body: { email: string; name: string; password: string }) =>
    request<User>('/auth/register', { method: 'POST', body: JSON.stringify(body) }),
  )

export const useLogout = () =>
  useSessionMutation((_: void) => request<void>('/auth/logout', { method: 'POST' }))
