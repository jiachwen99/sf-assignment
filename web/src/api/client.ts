import type { Blocker, Todo } from '../types'

type ErrorBody = {
  error?: string
  fields?: Record<string, string>
  current?: Todo
  blockers?: Blocker[]
}

// Mirrors the server's error body so callers can react to the shape rather
// than parsing a message string.
export class ApiError extends Error {
  status: number
  fields?: Record<string, string>
  current?: Todo
  blockers?: Blocker[]

  constructor(status: number, body: ErrorBody) {
    super(body.error ?? `request failed with ${status}`)
    this.status = status
    this.fields = body.fields
    this.current = body.current
    this.blockers = body.blockers
  }

  // Both a stale version and a blocked transition come back as 409: the request
  // is well formed and would be legal at another moment. `current` and
  // `blockers` say which it was.
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
