import type { ListQuery, SortDir, SortField } from '../types'

/*
 * The list state lives in the URL.
 *
 * That is what makes a filtered view shareable, what makes the back button do
 * something sensible, and what lets a test reach a state without clicking
 * through to it.
 *
 * Reading and writing are driven by the same two lists. The first version
 * spelled the readable keys out by hand and the writer took whatever was on the
 * object, so the two drifted the moment a key was added: `overdue` and
 * `recurring` were written to the URL and then dropped on the way back in, and
 * reloading a view silently returned you to all tasks.
 */

export const DEFAULT_SORT: SortField = 'created'
export const DEFAULT_DIR: SortDir = 'desc'

// Repeatable, because a status filter can hold several values at once.
const multi = ['status', 'priority'] as const

const single = ['name', 'dueFrom', 'dueTo', 'blocked', 'recurring', 'overdue'] as const

export function queryFromURL(search: string): ListQuery {
  const params = new URLSearchParams(search)
  const query: ListQuery = {
    sort: (params.get('sort') as SortField) ?? DEFAULT_SORT,
    dir: (params.get('dir') as SortDir) ?? DEFAULT_DIR,
  }

  for (const key of multi) {
    const values = params.getAll(key)
    if (values.length > 0) query[key] = values
  }
  for (const key of single) {
    const value = params.get(key)
    if (value) query[key] = value
  }
  return query
}

// The defaults are left out, so an untouched list has a clean URL and a shared
// one carries only what was actually chosen.
export function urlFromQuery(query: ListQuery): string {
  const params = new URLSearchParams()

  for (const key of multi) {
    query[key]?.forEach((v) => params.append(key, v))
  }
  for (const key of single) {
    if (query[key]) params.set(key, query[key])
  }
  if (query.sort && query.sort !== DEFAULT_SORT) params.set('sort', query.sort)
  if (query.dir && query.dir !== DEFAULT_DIR) params.set('dir', query.dir)

  const search = params.toString()
  return search ? `?${search}` : window.location.pathname
}
