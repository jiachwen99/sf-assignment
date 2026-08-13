import { useEffect, useState } from 'react'
import { ApiError } from '../api/client'
import { useCreateTodo, useDeleteTodo, useRefreshTodos, useUpdateTodo } from '../api/todos'
import { priorityOptions, statusOptions } from '../lib/format'
import type { Todo, TodoInput } from '../types'

/*
 * A panel beside the list, not a drawer over it.
 *
 * Inspecting a task usually means comparing it to its neighbours, and both need
 * the list to stay on screen. It only becomes an overlay when the viewport is
 * too narrow to hold both.
 */

// The two ways the server can refuse a write on a row you were looking at. The
// store goes to the trouble of telling them apart, so the panel does too: one
// has a version to move to, the other has nothing left to edit.
type Rejection = { kind: 'conflict'; current: Todo } | { kind: 'gone' }

const blank: TodoInput = {
  name: '',
  description: '',
  dueDate: null,
  status: 'not_started',
  priority: 'medium',
}

// One control vocabulary, so a select and an input line up on the same row.
const control =
  'h-8 w-full rounded-md border border-rule-firm bg-canvas px-2 text-[13px] text-ink transition-colors hover:border-ink-faint focus:border-action focus:outline-none'

const toInput = (t: Todo): TodoInput => ({
  name: t.name,
  description: t.description,
  dueDate: t.dueDate,
  status: t.status,
  priority: t.priority,
})

// datetime-local wants "YYYY-MM-DDTHH:mm" in local time, not an ISO instant.
const toLocalInput = (iso: string | null) => {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function TaskDetail({ todo, onClose }: { todo: Todo | 'new'; onClose: () => void }) {
  const isNew = todo === 'new'
  const existing = isNew ? null : todo

  // The row this form was built from. It starts as the one you clicked and
  // moves only when you accept a newer one, which is what makes the version
  // sent with a save the version you actually looked at.
  const [base, setBase] = useState<Todo | null>(existing)
  const [form, setForm] = useState<TodoInput>(existing ? toInput(existing) : blank)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [rejected, setRejected] = useState<Rejection | null>(null)

  const create = useCreateTodo()
  const update = useUpdateTodo()
  const remove = useDeleteTodo()
  const refreshTodos = useRefreshTodos()

  // Keyed on which task is open, not on the object. The list refetches after
  // every write, and resetting on a new object identity would throw away
  // whatever was half typed at the time.
  const openID = isNew ? 'new' : todo.id
  useEffect(() => {
    setBase(existing)
    setForm(existing ? toInput(existing) : blank)
    setErrors({})
    setRejected(null)
  }, [openID])

  // A rejected write is not a failure to report and move on from: the edits are
  // still in the form and still worth something, so the panel stays as it is
  // and the banner offers whatever choice is left.
  const handle = (err: unknown) => {
    if (!(err instanceof ApiError)) throw err
    if (err.fields) return setErrors(err.fields)
    if (err.isConflict && err.current) return setRejected({ kind: 'conflict', current: err.current })
    if (err.status === 404) return setRejected({ kind: 'gone' })
    throw err
  }

  const save = async () => {
    setErrors({})
    try {
      if (base) await update.mutateAsync({ id: base.id, version: base.version, input: form })
      else await create.mutateAsync(form)
      onClose()
    } catch (err) {
      handle(err)
    }
  }

  const destroy = async () => {
    if (!base) return
    try {
      await remove.mutateAsync({ id: base.id, version: base.version })
      onClose()
    } catch (err) {
      handle(err)
    }
  }

  return (
    <aside
      aria-label={isNew ? 'New task' : 'Task detail'}
      className="flex h-full w-full flex-col overflow-y-auto border-l border-rule bg-canvas"
    >
      <header className="sticky top-0 z-10 flex items-start gap-2 border-b border-rule bg-canvas px-4 py-3">
        {/* A textarea rather than an input: an input cannot wrap, so a long
            name is cut off mid-word. */}
        <textarea
          rows={1}
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.currentTarget.value })}
          onKeyDown={(e) => e.key === 'Enter' && e.preventDefault()}
          placeholder="Task name"
          aria-label="Task name"
          data-testid="todo-name"
          className="min-w-0 flex-1 resize-none border-0 bg-transparent p-0 text-[15px] leading-snug font-medium text-ink outline-none placeholder:text-ink-faint"
        />
        <button
          type="button"
          onClick={onClose}
          aria-label="Close panel"
          className="-mr-1 grid size-7 shrink-0 place-items-center rounded-md text-ink-faint transition-colors hover:bg-sunk hover:text-ink"
        >
          <svg viewBox="0 0 14 14" className="size-3.5" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden>
            <path d="m3 3 8 8M11 3l-8 8" strokeLinecap="round" />
          </svg>
        </button>
      </header>

      {errors.name && <p className="px-4 pt-2 text-[12px] text-late">{errors.name}</p>}

      {rejected && (
        <div
          data-testid="conflict-banner"
          className="mx-4 mt-3 rounded-md border border-halt-edge bg-halt-wash p-3"
        >
          {rejected.kind === 'conflict' ? (
            <>
              <p className="text-[13px] font-medium text-ink">Someone else changed this task</p>
              <p className="mt-1 text-[12px] text-ink-soft">
                Your edit was not saved. It now reads &ldquo;{rejected.current.name}&rdquo;.
              </p>
              {/* Load, not merge. Guessing which side of a field to keep is how
                  you lose the half nobody looked at. */}
              <button
                type="button"
                data-testid="conflict-reload"
                onClick={() => {
                  setBase(rejected.current)
                  setForm(toInput(rejected.current))
                  setRejected(null)
                }}
                className="mt-2 rounded-md bg-ink px-2.5 py-1 text-[12px] font-medium text-canvas transition-colors hover:bg-ink-soft"
              >
                Load the current version
              </button>
            </>
          ) : (
            <>
              <p className="text-[13px] font-medium text-ink">This task has been deleted</p>
              <p className="mt-1 text-[12px] text-ink-soft">
                Someone else removed it, so your edit has nowhere to go.
              </p>
              <button
                type="button"
                data-testid="conflict-close"
                onClick={() => {
                  refreshTodos()
                  onClose()
                }}
                className="mt-2 rounded-md bg-ink px-2.5 py-1 text-[12px] font-medium text-canvas transition-colors hover:bg-ink-soft"
              >
                Close
              </button>
            </>
          )}
        </div>
      )}

      <section className="border-t border-rule px-4 py-3.5">
        <h3 className="mb-2.5 text-[11px] font-medium tracking-wide text-ink-soft uppercase">Details</h3>

        <textarea
          value={form.description}
          onChange={(e) => setForm({ ...form, description: e.currentTarget.value })}
          placeholder="Add a description"
          aria-label="Description"
          rows={3}
          className="w-full resize-y rounded-md border border-rule-firm bg-canvas p-2 text-[13px] text-ink transition-colors hover:border-ink-faint focus:border-action focus:outline-none placeholder:text-ink-faint"
        />

        <div className="mt-3 grid grid-cols-2 gap-3">
          <label className="block">
            <span className="mb-1 block text-[12px] text-ink-soft">Status</span>
            <select
              value={form.status}
              onChange={(e) => setForm({ ...form, status: e.currentTarget.value as TodoInput['status'] })}
              data-testid="todo-status"
              className={control}
            >
              {statusOptions.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>

          <label className="block">
            <span className="mb-1 block text-[12px] text-ink-soft">Priority</span>
            <select
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: e.currentTarget.value as TodoInput['priority'] })}
              data-testid="todo-priority"
              className={control}
            >
              {priorityOptions.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
        </div>

        <label className="mt-3 block">
          <span className="mb-1 block text-[12px] text-ink-soft">Due</span>
          <input
            type="datetime-local"
            value={toLocalInput(form.dueDate)}
            onChange={(e) =>
              setForm({
                ...form,
                dueDate: e.currentTarget.value ? new Date(e.currentTarget.value).toISOString() : null,
              })
            }
            data-testid="todo-due"
            className={control}
          />
        </label>
      </section>

      <footer className="mt-auto flex items-center gap-2 border-t border-rule px-4 py-3">
        {base && (
          <button
            type="button"
            onClick={destroy}
            data-testid="todo-delete"
            className="rounded-md px-2.5 py-1.5 text-[13px] text-ink-soft transition-colors hover:bg-sunk hover:text-late"
          >
            Delete
          </button>
        )}
        <button
          type="button"
          onClick={save}
          data-testid="todo-save"
          className="ml-auto rounded-md bg-action px-3 py-1.5 text-[13px] font-medium text-white transition-colors hover:bg-action-hover"
        >
          {isNew ? 'Create task' : 'Save changes'}
        </button>
      </footer>
    </aside>
  )
}
