import { useEffect, useRef, useState } from 'react'

import { request } from './api/client'
import { useCounts, useRestoreTodo, useTodos, useTrash } from './api/todos'
import { AccountMenu } from './components/AccountMenu'
import { BulkBar } from './components/BulkBar'
import { EmptyState } from './components/EmptyState'
import { FilterRow } from './components/FilterRow'
import { TaskDetail } from './components/TaskDetail'
import { TaskList, TaskListSkeleton } from './components/TaskList'
import { TrashList } from './components/TrashList'
import { ViewRail } from './components/ViewRail'
import { Button } from './components/ui/Button'
import { useEvents } from './hooks/useEvents'
import { useInfiniteScroll } from './hooks/useInfiniteScroll'
import { DEFAULT_DIR, DEFAULT_SORT, queryFromURL, urlFromQuery } from './lib/url'
import { queryForView, viewFromQuery, viewTitle } from './lib/views'
import type { ListQuery, SortDir, SortField, Todo, ViewId } from './types'

// 'new' rather than a second piece of state: a task cannot be both open for
// editing and being created, and one value makes that unrepresentable.
type Selection = Todo | 'new' | null

export function App() {
  const [query, setQuery] = useState<ListQuery>(() => queryFromURL(window.location.search))
  const [inTrash, setInTrash] = useState(false)
  const [selection, setSelection] = useState<Selection>(null)
  const opened = useRef(0)
  const openPanel = (next: Selection) => {
    opened.current += 1
    setSelection(next)
  }
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  const { data, isPending, error, hasNextPage, isFetchingNextPage, fetchNextPage } = useTodos(query)
  const { data: counts } = useCounts()
  const { data: trash } = useTrash()
  const restore = useRestoreTodo()

  const todos = data?.pages.flatMap((p) => p.items)
  const view = viewFromQuery(query)

  const sentinel = useInfiniteScroll(Boolean(hasNextPage) && !isFetchingNextPage, fetchNextPage)

  // Anybody else's change refreshes what is on screen, without a reload.
  useEvents()

  useEffect(() => {
    window.history.replaceState(null, '', urlFromQuery(query))
  }, [query])

  const open =
    selection === 'new' || selection === null
      ? selection
      : (todos?.find((t) => t.id === selection.id) ?? selection)

  const closeWhenUntouched = (generation: number) => () => {
    if (opened.current === generation) setSelection(null)
  }

  const showTrash = (on: boolean) => {
    setInTrash(on)
    // The panel edits a live task, and nothing in the trash is editable.
    setSelection(null)
  }

  const goToView = (next: ViewId) => {
    setQuery(queryForView(next, query))
    showTrash(false)
  }

  const sortBy = (sort: SortField, dir: SortDir) => setQuery({ ...query, sort, dir })

  const select = (ids: number[], checked: boolean) =>
    setSelectedIds((current) => {
      const next = new Set(current)
      ids.forEach((id) => (checked ? next.add(id) : next.delete(id)))
      return next
    })

  // Held by id and resolved against the list, so a selected row that has since
  // moved out of view is not acted on with a stale version.
  const selected = todos?.filter((t) => selectedIds.has(t.id)) ?? []

  return (
    <div className="flex h-screen bg-canvas text-ink">
      {/* The rail only appears once there is room for it and the table both.
          Below that it would be taking width from the thing it navigates. */}
      <aside className="hidden w-52 shrink-0 overflow-y-auto border-r border-rule xl:block">
        <ViewRail active={view} counts={counts} onSelect={goToView} />
      </aside>

      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <header className="flex items-center gap-3 border-b border-rule px-4 py-3">
          <h1 className="text-[15px] font-medium">{inTrash ? 'Trash' : viewTitle(view)}</h1>
          {/* From the counts rather than from the fetched array: the trash is
              capped at the recent end, so its length is how much arrived, not
              how much is there. */}
          {inTrash && <span className="tabular text-[13px] text-ink-faint">{counts?.trash}</span>}

          {/* A link rather than a view in its own right. The trash is somewhere
              you go to undo something, not somewhere you work. */}
          <Button
            variant="quiet"
            size="sm"
            className="ml-auto"
            onClick={() => showTrash(!inTrash)}
            data-testid="toggle-trash"
          >
            {inTrash ? 'Back to tasks' : `Trash${counts?.trash ? ` (${counts.trash})` : ''}`}
          </Button>

          <AccountMenu />

          {!inTrash && (
            <Button variant="primary" onClick={() => openPanel('new')} data-testid="new-todo">
              New task
            </Button>
          )}
        </header>

        {/* The same views as a strip once the rail no longer fits. */}
        {!inTrash && (
          <div className="xl:hidden">
            <ViewRail active={view} counts={counts} onSelect={goToView} orientation="horizontal" />
          </div>
        )}

        {!inTrash && <FilterRow query={query} onChange={setQuery} />}

        {!inTrash && selected.length > 0 && (
          <BulkBar selected={selected} onClear={() => setSelectedIds(new Set())} />
        )}

        <div className="flex-1 overflow-y-auto">
          {inTrash ? (
            <>
              <TrashList todos={trash ?? []} onRestore={(id) => restore.mutate(id)} />
              {/* Said out loud rather than left to be inferred from a list that
                  stops. Anything older than this is still in the database. */}
              {trash && counts && counts.trash > trash.length && (
                <p className="px-4 py-3 text-[13px] text-ink-faint" data-testid="trash-capped">
                  Showing the {trash.length} most recently deleted of {counts.trash}.
                </p>
              )}
            </>
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
                  <EmptyState onCreate={() => openPanel('new')} />
                ) : (
                  <>
                    <TaskList
                      todos={todos}
                      activeId={open && open !== 'new' ? open.id : null}
                      sort={query.sort ?? DEFAULT_SORT}
                      dir={query.dir ?? DEFAULT_DIR}
                      selected={selectedIds}
                      onSelect={select}
                      onSort={sortBy}
                      onOpen={openPanel}
                    />
                    <div ref={sentinel} aria-hidden />
                    {hasNextPage && (
                      <div className="flex justify-center py-4">
                        <Button
                          onClick={() => fetchNextPage()}
                          disabled={isFetchingNextPage}
                          data-testid="next-page"
                        >
                          {isFetchingNextPage ? 'Loading…' : 'Load more'}
                        </Button>
                      </div>
                    )}
                  </>
                ))}
            </>
          )}
        </div>
      </main>

      {open && (
        <div className="w-[340px] shrink-0">
          <TaskDetail
            todo={open}
            onClose={closeWhenUntouched(opened.current)}
            onCreated={openPanel}
            // The chain and the history are also how you navigate, and both can
            // point outside the current view: a completed predecessor is not in
            // the recurring list. Fetching it is the difference between a link
            // and a link that does nothing.
            onOpenTask={async (id) => {
              const known = todos?.find((t) => t.id === id)
              openPanel(known ?? (await request<Todo>(`/todos/${id}`)))
            }}
          />
        </div>
      )}
    </div>
  )
}
