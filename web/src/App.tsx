import { useState } from 'react'

import { useTodos } from './api/todos'
import { EmptyState } from './components/EmptyState'
import { TaskDetail } from './components/TaskDetail'
import { TaskList, TaskListSkeleton } from './components/TaskList'
import { Button } from './components/ui/Button'
import type { Todo } from './types'

// 'new' rather than a second piece of state: a task cannot be both open for
// editing and being created, and one value makes that unrepresentable.
type Selection = Todo | 'new' | null

export function App() {
  const { data: todos, isPending, error } = useTodos()
  const [selection, setSelection] = useState<Selection>(null)

  // The list is the source of truth, so a selected task follows its own edits
  // instead of holding the copy that was clicked.
  const open =
    selection === 'new' || selection === null
      ? selection
      : (todos?.find((t) => t.id === selection.id) ?? null)

  return (
    <div className="flex h-screen bg-canvas text-ink">
      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <header className="flex items-center gap-3 border-b border-rule px-4 py-3">
          <h1 className="text-[15px] font-medium">Tasks</h1>
          {todos && <span className="tabular text-[13px] text-ink-faint">{todos.length}</span>}
          <Button
            variant="primary"
            className="ml-auto"
            onClick={() => setSelection('new')}
            data-testid="new-todo"
          >
            New task
          </Button>
        </header>

        <div className="flex-1 overflow-y-auto">
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
        </div>
      </main>

      {open && (
        <div className="w-[340px] shrink-0">
          <TaskDetail
            todo={open}
            onClose={() => setSelection(null)}
            // The chain is also how you navigate it: clicking a node opens that
            // task in the same panel.
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
