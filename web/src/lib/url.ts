import type { ListQuery, SortDir, SortField } from '../types'

/*
 * The list state lives in the URL.
 *
 * That is what makes a filtered view shareable, what makes the back button do
 * something sensible, and what lets a test reach a state without clicking
 * through to it.
 */

export const DEFAULT_SORT: SortField = 'created'
export const DEFAULT_DIR: SortDir = 'desc'

const multi = ['status', 'priority'] as const

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
  for (const key of ['name', 'dueFrom', 'dueTo', 'blocked'] as const) {
    const value = params.get(key)
    if (value) query[key] = value
  }
  return query
}

// The defaults are left out, so an untouched list has a clean URL and a shared
// one carries only what was actually chosen.
export function urlFromQuery(query: ListQuery): string {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(query)) {
    if (key === 'sort' && value === DEFAULT_SORT) continue
    if (key === 'dir' && value === DEFAULT_DIR) continue
    if (Array.isArray(value)) value.forEach((v) => params.append(key, v))
    else if (value) params.set(key, value)
  }
  const search = params.toString()
  return search ? `?${search}` : window.location.pathname
}
