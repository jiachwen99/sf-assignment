import type { Blocker, Todo } from '../../types'
import { Button } from '../ui/Button'
import { Notice } from '../ui/Notice'

/*
 * The three ways the server refuses a write on a task you were looking at.
 *
 * The store goes to the trouble of telling them apart, so the panel does too:
 * one has a version to move to, one has work to go and finish, and one has
 * nothing left to edit at all.
 */
export type Rejection =
  | { kind: 'conflict'; current: Todo }
  | { kind: 'blocked'; blockers: Blocker[] }
  | { kind: 'gone' }

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
      ) : rejection.kind === 'blocked' ? (
        // The names, not the count. A count says you are stuck; the names say
        // what to go and finish.
        <Notice title="This task is waiting on unfinished work" data-testid="blocked-banner">
          <p className="mt-1 text-[12px] text-ink-soft">
            Blocked by {rejection.blockers.map((b) => b.name).join(', ')}. Only completing a task
            releases what waits on it.
          </p>
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
