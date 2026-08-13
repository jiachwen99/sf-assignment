import { useEffect, useState } from 'react'

import { MIN_SEARCH } from '../constants'
import { priorityOptions, statusOptions } from '../lib/format'
import type { ListQuery } from '../types'
import { Button } from './ui/Button'
import { Input, Select } from './ui/Control'

/*
 * Refinement, not navigation. Everything here is optional, so the row stays
 * quiet until something is set.
 *
 * Every filter the brief names gets a control. One reachable only by editing
 * the URL is not a missing feature, it is a missing control, which is harder
 * for anyone to notice.
 *
 * Widths live on wrappers rather than on the controls. The controls fill what
 * they are given, and a width class passed to one would be arguing with the
 * `w-full` in the primitive over which utility the cascade emits last.
 */

// A date input gives a day; a filter needs an instant. From is the start of the
// day and to is the end of it, so "due to the 3rd" includes the 3rd.
const startOfDay = (date: string) => (date ? new Date(`${date}T00:00`).toISOString() : undefined)
const endOfDay = (date: string) => (date ? new Date(`${date}T23:59:59`).toISOString() : undefined)
const asDate = (iso?: string) => (iso ? new Date(iso).toISOString().slice(0, 10) : '')

export function FilterRow({
  query,
  onChange,
}: {
  query: ListQuery
  onChange: (next: ListQuery) => void
}) {
  const [term, setTerm] = useState(query.name ?? '')

  // Typing should not fire a request per keystroke, and anything shorter than
  // the minimum never reaches the trigram index at all.
  useEffect(() => {
    const trimmed = term.trim()
    if (trimmed === (query.name ?? '')) return
    const id = setTimeout(() => {
      onChange({ ...query, name: trimmed.length >= MIN_SEARCH ? trimmed : undefined })
    }, 250)
    return () => clearTimeout(id)
  }, [term])

  const set = (patch: Partial<ListQuery>) => onChange({ ...query, ...patch })

  const active =
    (query.status?.length ?? 0) +
    (query.priority?.length ?? 0) +
    (query.name ? 1 : 0) +
    (query.blocked ? 1 : 0) +
    (query.dueFrom ? 1 : 0) +
    (query.dueTo ? 1 : 0)

  const tooShort = term.trim().length > 0 && term.trim().length < MIN_SEARCH

  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-rule px-4 py-2.5">
      <div className="relative w-52">
        <Input
          value={term}
          onChange={(e) => setTerm(e.currentTarget.value)}
          placeholder="Search names"
          aria-label="Search tasks by name"
          data-testid="filter-name"
        />
        {tooShort && (
          <p className="absolute top-full left-0 mt-1 text-[11px] text-ink-faint">
            Keep typing, {MIN_SEARCH} characters minimum
          </p>
        )}
      </div>

      {/* Native selects rather than invented dropdowns. They are keyboard
          accessible for free and behave the way people already expect. */}
      <div className="w-36">
        <Select
          value={query.status?.[0] ?? ''}
          onChange={(e) =>
            set({ status: e.currentTarget.value ? [e.currentTarget.value] : undefined })
          }
          aria-label="Filter by status"
          data-testid="filter-status"
        >
          <option value="">Any status</option>
          {statusOptions.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
      </div>

      <div className="w-36">
        <Select
          value={query.priority?.[0] ?? ''}
          onChange={(e) =>
            set({ priority: e.currentTarget.value ? [e.currentTarget.value] : undefined })
          }
          aria-label="Filter by priority"
          data-testid="filter-priority"
        >
          <option value="">Any priority</option>
          {priorityOptions.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </Select>
      </div>

      <div className="w-40">
        <Select
          value={query.blocked ?? ''}
          onChange={(e) => set({ blocked: e.currentTarget.value || undefined })}
          aria-label="Filter by dependency status"
          data-testid="filter-blocked"
        >
          <option value="">Blocked or not</option>
          <option value="true">Blocked</option>
          <option value="false">Not blocked</option>
        </Select>
      </div>

      <div className="flex items-center gap-1.5 text-[12px] text-ink-soft">
        Due
        <div className="w-36">
          <Input
            type="date"
            value={asDate(query.dueFrom)}
            onChange={(e) => set({ dueFrom: startOfDay(e.currentTarget.value) })}
            aria-label="Due on or after"
            data-testid="filter-due-from"
          />
        </div>
        to
        <div className="w-36">
          <Input
            type="date"
            value={asDate(query.dueTo)}
            onChange={(e) => set({ dueTo: endOfDay(e.currentTarget.value) })}
            aria-label="Due on or before"
            data-testid="filter-due-to"
          />
        </div>
      </div>

      {active > 0 && (
        <Button
          variant="quiet"
          size="sm"
          onClick={() => {
            setTerm('')
            onChange({ sort: query.sort, dir: query.dir })
          }}
          data-testid="clear-filters"
        >
          Clear {active}
        </Button>
      )}
    </div>
  )
}
