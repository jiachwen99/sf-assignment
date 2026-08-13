import { useEffect, useState } from 'react'
import { ApiError } from '../api/client'
import { useCreateTodo, useDeleteTodo, useUpdateTodo } from '../api/todos'
import { priorityOptions, statusOptions } from '../lib/format'
import type { Todo, TodoInput } from '../types'

/*
 * A panel beside the list, not a drawer over it.
 *
 * Inspecting a task usually means comparing it to its neighbours, and both need
 * the list to stay on screen. It only becomes an overlay when the viewport is
 * too narrow to hold both.
 */

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

  const [form, setForm] = useState<TodoInput>(existing ? toInput(existing) : blank)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const create = useCreateTodo()
  const update = useUpdateTodo()
  const remove = useDeleteTodo()

  useEffect(() => {
    setForm(existing ? toInput(existing) : blank)
    setErrors({})
  }, [todo])

  const save = async () => {
    setErrors({})
    try {
      if (existing) await update.mutateAsync({ id: existing.id, input: form })
      else await create.mutateAsync(form)
      onClose()
    } catch (err) {
      if (err instanceof ApiError && err.fields) setErrors(err.fields)
      else throw err
    }
  }

  const destroy = async () => {
    if (!existing) return
    await remove.mutateAsync(existing.id)
    onClose()
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
        {existing && (
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
