import { useState } from 'react'

import { useBulk } from '../api/todos'
import type { BulkResult, Todo } from '../types'
import { Button } from './ui/Button'
import { Notice } from './ui/Notice'

/*
 * Appears only when there is a selection, anchored directly above the list it
 * acts on rather than parked in the header.
 *
 * Results report per item, because one stale or blocked task should not discard
 * the rest of the batch. That means this has to be able to say "forty-nine
 * worked and here is the one that did not", which a toast cannot.
 */
export function BulkBar({
  selected,
  onClear,
}: {
  selected: Todo[]
  onClear: () => void
}) {
  const [refused, setRefused] = useState<BulkResult[]>([])

  const complete = useBulk('complete')
  const archive = useBulk('archive')
  const busy = complete.isPending || archive.isPending

  const run = async (action: typeof complete) => {
    const results = await action.mutateAsync(
      selected.map((t) => ({ id: t.id, version: t.version })),
    )
    const failed = results.filter((r) => !r.ok)
    setRefused(failed)

    // Everything worked, so there is nothing left to act on. A selection that
    // survives a completed batch is a selection of rows that have all moved.
    if (failed.length === 0) onClear()
  }

  const names = new Map(selected.map((t) => [t.id, t.name]))

  return (
    <div className="border-b border-rule bg-raised px-4 py-2.5">
      <div className="flex flex-wrap items-center gap-2">
        <span className="tabular text-[13px] font-medium text-ink">
          {selected.length} selected
        </span>

        <div className="ml-auto flex items-center gap-1.5">
          <Button
            variant="primary"
            size="sm"
            onClick={() => run(complete)}
            disabled={busy}
            data-testid="bulk-complete"
          >
            Complete
          </Button>
          <Button size="sm" onClick={() => run(archive)} disabled={busy} data-testid="bulk-archive">
            Archive
          </Button>
          <Button variant="quiet" size="sm" onClick={onClear} data-testid="bulk-clear">
            Clear
          </Button>
        </div>
      </div>

      {refused.length > 0 && (
        <div className="mt-2.5">
          <Notice
            title={`${refused.length} of ${selected.length} could not be done`}
            data-testid="bulk-refused"
          >
            {/* Named, not counted. Which ones failed and why is the only part
                of this anybody can act on. */}
            <ul className="mt-1 space-y-0.5">
              {refused.map((result) => (
                <li key={result.id} className="text-[12px] text-ink-soft">
                  <span className="text-ink">{names.get(result.id) ?? `Task ${result.id}`}</span>
                  {' — '}
                  {result.error}
                  {result.blockers?.length ? (
                    <>: waiting on {result.blockers.map((b) => b.name).join(', ')}</>
                  ) : null}
                </li>
              ))}
            </ul>
          </Notice>
        </div>
      )}
    </div>
  )
}
