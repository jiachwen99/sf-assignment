import { useState } from 'react'

import { useRestoreTodo, useTodos, useTrash } from './api/todos'
import { EmptyState } from './components/EmptyState'
import { TaskDetail } from './components/TaskDetail'
import { TaskList, TaskListSkeleton } from './components/TaskList'
import { TrashList } from './components/TrashList'
import { Button } from './components/ui/Button'
import type { Todo } from './types'

// 'new' rather than a second piece of state: a task cannot be both open for
// editing and being created, and one value makes that unrepresentable.
type Selection = Todo | 'new' | null

export function App() {
  const [inTrash, setInTrash] = useState(false)
  const [selection, setSelection] = useState<Selection>(null)

  const { data: todos, isPending, error } = useTodos()
  const { data: trash } = useTrash()
  const restore = useRestoreTodo()

  // The list is the source of truth, so a selected task follows its own edits
  // instead of holding the copy that was clicked.
  const open =
    selection === 'new' || selection === null
      ? selection
      : (todos?.find((t) => t.id === selection.id) ?? null)

  const showTrash = (on: boolean) => {
    setInTrash(on)
    // The panel edits a live task, and nothing in the trash is editable.
    setSelection(null)
  }

  return (
    <div className="flex h-screen bg-canvas text-ink">
      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <header className="flex items-center gap-3 border-b border-rule px-4 py-3">
          <h1 className="text-[15px] font-medium">{inTrash ? 'Trash' : 'Tasks'}</h1>
          <span className="tabular text-[13px] text-ink-faint">
            {inTrash ? trash?.length : todos?.length}
          </span>

          {/* A link rather than a view in its own right. The trash is somewhere
              you go to undo something, not somewhere you work. */}
          <Button
            variant="quiet"
            size="sm"
            className="ml-auto"
            onClick={() => showTrash(!inTrash)}
            data-testid="toggle-trash"
          >
            {inTrash ? 'Back to tasks' : `Trash${trash?.length ? ` (${trash.length})` : ''}`}
          </Button>

          {!inTrash && (
            <Button variant="primary" onClick={() => setSelection('new')} data-testid="new-todo">
              New task
            </Button>
          )}
        </header>

        <div className="flex-1 overflow-y-auto">
          {inTrash ? (
            <TrashList todos={trash ?? []} onRestore={(id) => restore.mutate(id)} />
          ) : (
            <>
              {isPending && <TaskListSkeleton />}
              {error && (
                <p className="px-4 py-6 text-[13px] text-late">
                  The task list could not be loaded. Check that the API is running, then reload.
                </p>
              )}
              {todos &&
                (todos.length === 0 ? (
                  <EmptyState onCreate={() => setSelection('new')} />
                ) : (
                  <TaskList
                    todos={todos}
                    activeId={open && open !== 'new' ? open.id : null}
                    onOpen={setSelection}
                  />
                ))}
            </>
          )}
        </div>
      </main>

      {open && (
        <div className="w-[340px] shrink-0">
          <TaskDetail
            todo={open}
            onClose={() => setSelection(null)}
            // The chain is also how you navigate it: clicking a node opens that
            // task in the same panel. A deleted node has nothing to open.
            onOpenTask={(id) => {
              const next = todos?.find((t) => t.id === id)
              if (next) setSelection(next)
            }}
          />
        </div>
      )}
    </div>
  )
}
