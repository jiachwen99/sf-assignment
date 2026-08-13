import { describeCreated } from '../lib/dates'
import type { Todo } from '../types'
import { Button } from './ui/Button'

/*
 * Deliberately not the task table.
 *
 * Restore is the only thing you can do to a deleted task, so reusing the list
 * would offer a row of controls that all refuse. What matters here is what the
 * task was called and when it went, which is two columns, not five.
 */
export function TrashList({
  todos,
  onRestore,
}: {
  todos: Todo[]
  onRestore: (id: number) => void
}) {
  if (todos.length === 0) {
    return (
      <div className="grid place-items-center px-6 py-20 text-center">
        <p className="text-[15px] text-ink">The trash is empty</p>
        <p className="mt-1 max-w-sm text-[13px] text-ink-soft">
          Deleted tasks wait here. Nothing is removed for good, and restoring one puts it back
          exactly where it was, links and all.
        </p>
      </div>
    )
  }

  return (
    <ul className="divide-y divide-rule">
      {todos.map((todo) => (
        <li key={todo.id} className="flex items-center gap-3 px-4 py-2.5 hover:bg-raised">
          <span className="min-w-0 flex-1 truncate text-[13px] text-ink-soft">{todo.name}</span>
          <span className="tabular shrink-0 text-[12px] text-ink-faint">
            created {describeCreated(todo.createdAt).toLowerCase()}
          </span>
          <Button size="sm" onClick={() => onRestore(todo.id)} data-testid="restore-button">
            Restore
          </Button>
        </li>
      ))}
    </ul>
  )
}
