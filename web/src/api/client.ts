import type { Todo } from '../types'

// Mirrors the server's error body so callers can react to the shape rather
// than parsing a message string.
export class ApiError extends Error {
  status: number
  fields?: Record<string, string>
  current?: Todo

  constructor(
    status: number,
    body: { error?: string; fields?: Record<string, string>; current?: Todo },
  ) {
    super(body.error ?? `request failed with ${status}`)
    this.status = status
    this.fields = body.fields
    this.current = body.current
  }

  get isConflict() {
    return this.status === 409
  }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })

  if (res.status === 204) return undefined as T
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, body)
  return body as T
}
