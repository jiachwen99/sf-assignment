import type { APIRequestContext } from '@playwright/test'

/*
 * The tests drive the browser for everything a user does. These helpers exist
 * only to reach a precondition, and to play the part of somebody else editing
 * the same task.
 *
 * Clicking your way to a precondition tests the same three screens over and
 * over and puts the failure a long way from the thing under test.
 */

export const API = process.env.API_URL ?? 'http://localhost:8080'

export type Task = {
  id: number
  name: string
  status: string
  priority: string
  dueDate: string | null
  recurUnit: string | null
  recurInterval: number | null
  version: number
}

async function json<T>(res: Awaited<ReturnType<APIRequestContext['post']>>, what: string): Promise<T> {
  if (!res.ok()) throw new Error(`${what} failed: ${res.status()} ${await res.text()}`)
  return res.json()
}

export async function createTask(
  request: APIRequestContext,
  body: Record<string, unknown>,
): Promise<Task> {
  return json(await request.post(`${API}/api/todos`, { data: body }), 'create')
}

// Sends the whole task back with the patch on top, because the API takes a
// complete representation rather than a partial one.
export async function updateTask(
  request: APIRequestContext,
  task: Task,
  patch: Record<string, unknown>,
): Promise<Task> {
  return json(
    await request.put(`${API}/api/todos/${task.id}`, {
      data: {
        name: task.name,
        description: '',
        dueDate: task.dueDate,
        status: task.status,
        priority: task.priority,
        recurUnit: task.recurUnit,
        recurInterval: task.recurInterval,
        version: task.version,
        ...patch,
      },
    }),
    'update',
  )
}

export async function completeTask(request: APIRequestContext, task: Task) {
  return json<{ completed: Task; spawned: Task | null }>(
    await request.post(`${API}/api/todos/${task.id}/complete?version=${task.version}`),
    'complete',
  )
}

export async function addDependency(request: APIRequestContext, todoId: number, dependsOnId: number) {
  await json(
    await request.post(`${API}/api/todos/${todoId}/dependencies`, { data: { dependsOnId } }),
    'link',
  )
}

export async function readTask(request: APIRequestContext, id: number): Promise<Task> {
  return json(await request.get(`${API}/api/todos/${id}`), 'read')
}

/*
 * A unique name per test, so the suite can run against a seeded database
 * without truncating it.
 *
 * Truncating would make the tests pass against an empty table and prove
 * nothing about the list at twenty thousand rows, which is the size the brief
 * actually cares about.
 */
export function unique(label: string) {
  return `e2e ${label} ${Date.now()}${Math.floor(Math.random() * 1000)}`
}

// Read from the API rather than through the dev server, because the count lives
// in the process that holds the subscriptions.
export async function subscriberCount(request: APIRequestContext): Promise<number> {
  const res = await request.get(`${API}/healthz`)
  return (await res.json()).subscribers
}
