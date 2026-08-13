import type { Todo } from '../../types'
import { Button } from '../ui/Button'
import { Notice } from '../ui/Notice'

/*
 * The two ways the server refuses a write on a row you were looking at.
 *
 * The store goes to the trouble of telling a stale version apart from a deleted
 * row, so the panel does too: one has a version to move to, the other has
 * nothing left to edit.
 */
export type Rejection = { kind: 'conflict'; current: Todo } | { kind: 'gone' }

export function RejectionNotice({
  rejection,
  onLoad,
  onClose,
}: {
  rejection: Rejection
  onLoad: (current: Todo) => void
  onClose: () => void
}) {
  return (
    <div className="mx-4 mt-3">
      {rejection.kind === 'conflict' ? (
        <Notice title="Someone else changed this task" data-testid="conflict-banner">
          <p className="mt-1 text-[12px] text-ink-soft">
            Your edit was not saved. It now reads &ldquo;{rejection.current.name}&rdquo;.
          </p>
          {/* Load, not merge. Guessing which side of each field to keep is how
              you lose the half nobody looked at. */}
          <Button
            variant="contrast"
            size="sm"
            className="mt-2"
            data-testid="conflict-reload"
            onClick={() => onLoad(rejection.current)}
          >
            Load the current version
          </Button>
        </Notice>
      ) : (
        <Notice title="This task has been deleted" data-testid="conflict-banner">
          <p className="mt-1 text-[12px] text-ink-soft">
            Someone else removed it, so your edit has nowhere to go.
          </p>
          <Button
            variant="contrast"
            size="sm"
            className="mt-2"
            data-testid="conflict-close"
            onClick={onClose}
          >
            Close
          </Button>
        </Notice>
      )}
    </div>
  )
}
