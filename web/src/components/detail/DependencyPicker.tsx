import { useState } from 'react'
import { useTodoSearch } from '../../api/todos'
import { MIN_SEARCH } from '../../constants'
import type { Blocker } from '../../types'
import { Input } from '../ui/Control'

/*
 * Queried by substring as you type, held back until three characters.
 *
 * The option list is every other task, so loading it into the browser would be
 * the same mistake as filtering in application memory: fine at ten rows,
 * hopeless at two hundred thousand.
 */
export function DependencyPicker({
  taskId,
  existing,
  onPick,
}: {
  taskId: number
  existing: Blocker[]
  onPick: (id: number) => void
}) {
  const [search, setSearch] = useState('')
  const { data: matches } = useTodoSearch(search, taskId)

  const offered = (matches ?? [])
    .filter((m) => !existing.some((d) => d.id === m.id))
    .slice(0, 8)

  return (
    <div className="relative mt-3">
      <Input
        value={search}
        onChange={(e) => setSearch(e.currentTarget.value)}
        placeholder={`Make this wait for… (${MIN_SEARCH} characters)`}
        aria-label="Search for a task to depend on"
        data-testid="dependency-search"
      />
      {search.trim().length >= MIN_SEARCH && offered.length > 0 && (
        <ul className="absolute z-20 mt-1 max-h-56 w-full overflow-y-auto rounded-md border border-rule-firm bg-canvas py-1 shadow-lg">
          {offered.map((m) => (
            <li key={m.id}>
              <button
                type="button"
                onClick={() => {
                  onPick(m.id)
                  setSearch('')
                }}
                className="w-full truncate px-2.5 py-1.5 text-left text-[13px] text-ink hover:bg-sunk"
              >
                {m.name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
