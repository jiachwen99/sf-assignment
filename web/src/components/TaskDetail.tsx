import { useEffect, useRef, useState } from 'react'

import { ApiError } from '../api/client'
import {
  useAddDependency,
  useCompleteTodo,
  useCreateTodo,
  useDeleteTodo,
  useDependencies,
  useRefreshTodos,
  useRemoveDependency,
  useUpdateTodo,
} from '../api/todos'
import { toLocalInput } from '../lib/dates'
import { priorityOptions, statusOptions } from '../lib/format'
import type { Priority, Status, Todo, TodoInput } from '../types'

import { DependencyChain } from './DependencyChain'
import { TaskHistory } from './TaskHistory'
import { DependencyPicker } from './detail/DependencyPicker'
import { RejectionNotice, type Rejection } from './detail/RejectionNotice'
import { RepeatsField } from './detail/RepeatsField'
import { TaskNameField } from './detail/TaskNameField'
import { Button, IconButton } from './ui/Button'
import { ConfirmDialog } from './ui/ConfirmDialog'
import { Field, Input, Select, TextArea } from './ui/Control'
import { CloseIcon } from './ui/icons'
import { Section } from './ui/Notice'

/*
 * A panel beside the list, not a drawer over it.
 *
 * Inspecting a task usually means comparing it to its neighbours, and both need
 * the list to stay on screen. It only becomes an overlay when the viewport is
 * too narrow to hold both.
 *
 * This file holds the state and the order of the sections. Each section is its
 * own component next door, because a panel that owns a form, a rejected write
 * and a schedule is three screens' worth of decisions in one scroll.
 */

const blank: TodoInput = {
  name: '',
  description: '',
  dueDate: null,
  status: 'not_started',
  priority: 'medium',
  recurUnit: null,
  recurInterval: null,
}

const toInput = (t: Todo): TodoInput => ({
  name: t.name,
  description: t.description,
  dueDate: t.dueDate,
  status: t.status,
  priority: t.priority,
  recurUnit: t.recurUnit,
  recurInterval: t.recurInterval,
})

export function TaskDetail({
  todo,
  onClose,
  onCreated,
  onOpenTask,
}: {
  todo: Todo | 'new'
  onClose: () => void
  onCreated: (todo: Todo) => void
  onOpenTask: (id: number) => void
}) {
  const isNew = todo === 'new'
  const existing = isNew ? null : todo

  // The row this form was built from. It starts as the one you clicked and
  // moves only when you accept a newer one, which is what makes the version
  // sent with a save the version you actually looked at.
  const [base, setBase] = useState<Todo | null>(existing)
  const [form, setForm] = useState<TodoInput>(existing ? toInput(existing) : blank)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [rejected, setRejected] = useState<Rejection | null>(null)
  const [confirming, setConfirming] = useState(false)

  const create = useCreateTodo()
  const update = useUpdateTodo()
  const remove = useDeleteTodo()
  const complete = useCompleteTodo()
  const addDep = useAddDependency()
  const dropDep = useRemoveDependency()
  const refreshTodos = useRefreshTodos()

  const { data: chain, isPending: chainPending } = useDependencies(base?.id ?? null)
  const dependencies = chain?.dependencies ?? []
  const dependents = chain?.dependents ?? []
  const blockers = dependencies.filter((d) => d.status !== 'completed')

  // Keyed on which task is open, not on the object. The list refetches after
  // every write, and resetting on a new object identity would throw away
  // whatever was half typed at the time.
  const openID = isNew ? 'new' : todo.id
  // A write resolves after the panel may have moved on, so its result is only
  // applied if the panel is still showing what it was written for.
  const showing = useRef<number | 'new'>(openID)
  useEffect(() => {
    showing.current = openID
    setBase(existing)
    setForm(existing ? toInput(existing) : blank)
    setErrors({})
    setRejected(null)
    setConfirming(false)
  }, [openID])

  const patch = (fields: Partial<TodoInput>) => setForm((f) => ({ ...f, ...fields }))

  // Completed is missing from the select until the task is, because finishing a
  // task is the Complete action below: it opens the next occurrence of a
  // repeating one, which setting a field cannot do. Reopening is an ordinary
  // edit, so the option stays visible afterwards.
  const statusChoices =
    form.status === 'completed' ? statusOptions : statusOptions.filter((o) => o.value !== 'completed')

  // A rejected write is not a failure to report and move on from: the edits are
  // still in the form and still worth something, so the panel stays as it is
  // and the notice offers whatever choice is left.
  const handle = (err: unknown) => {
    if (!(err instanceof ApiError)) throw err
    if (err.fields) return setErrors(err.fields)
    if (err.isConflict && err.current) return setRejected({ kind: 'conflict', current: err.current })
    if (err.isConflict && err.blockers) return setRejected({ kind: 'blocked', blockers: err.blockers })
    if (err.status === 404) return setRejected({ kind: 'gone' })
    throw err
  }

  // Linking clears only its own error, so a refused link does not wipe a field
  // error sitting above it in the form.
  const link = async (pending: Promise<unknown>) => {
    setErrors((e) => ({ ...e, dependsOnId: '' }))
    try {
      await pending
    } catch (err) {
      handle(err)
    }
  }

  const save = async () => {
    setErrors({})
    try {
      if (base) {
        const editing = openID
        const saved = await update.mutateAsync({ id: base.id, version: base.version, input: form })
        if (showing.current !== editing) return
        setBase(saved)
        setForm(toInput(saved))
      } else {
        // Hand the new task to the list so the panel is editing it rather than
        // still offering to create another one.
        onCreated(await create.mutateAsync(form))
      }
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

  const finish = async () => {
    if (!base) return
    try {
      const editing = openID
      const { completed } = await complete.mutateAsync({ id: base.id, version: base.version })
      if (showing.current !== editing) return
      setBase(completed)
      setForm(toInput(completed))
    } catch (err) {
      handle(err)
    }
  }

  const busy = create.isPending || update.isPending || remove.isPending || complete.isPending

  return (
    <aside
      aria-label={isNew ? 'New task' : 'Task detail'}
      className="flex h-full w-full flex-col overflow-y-auto border-l border-rule bg-canvas"
    >
      <header className="sticky top-0 z-10 flex justify-end border-b border-rule bg-canvas px-4 py-2">
        <IconButton onClick={onClose} aria-label="Close panel" className="-mr-1">
          <CloseIcon />
        </IconButton>
      </header>

      {rejected && (
        <RejectionNotice
          rejection={rejected}
          onLoad={(current) => {
            setBase(current)
            setForm(toInput(current))
            setRejected(null)
          }}
          onClose={() => {
            refreshTodos()
            onClose()
          }}
        />
      )}

      <Section title="">
        <div className="space-y-3">
          <TaskNameField
            form={form}
            onChange={(name) => patch({ name })}
            error={errors.name}
            autoFocus={isNew}
          />

          <Field label="Description">
            <TextArea
              value={form.description}
              onChange={(e) => patch({ description: e.currentTarget.value })}
              placeholder="Add a description"
              rows={3}
              aria-label="Description"
            />
          </Field>

          <div className="grid grid-cols-2 gap-3">
            <Field label="Status">
              <Select
                value={form.status}
                onChange={(e) => patch({ status: e.currentTarget.value as Status })}
                data-testid="todo-status"
              >
                {statusChoices.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label="Priority">
              <Select
                value={form.priority}
                onChange={(e) => patch({ priority: e.currentTarget.value as Priority })}
                data-testid="todo-priority"
              >
                {priorityOptions.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </Select>
            </Field>
          </div>

          <Field label="Due">
            <Input
              type="datetime-local"
              value={toLocalInput(form.dueDate)}
              onChange={(e) =>
                patch({
                  dueDate: e.currentTarget.value
                    ? new Date(e.currentTarget.value).toISOString()
                    : null,
                })
              }
              data-testid="todo-due"
            />
          </Field>
        </div>
      </Section>

      <Section title="Repeats">
        <RepeatsField form={form} onChange={patch} />
      </Section>

      {base && (
        <Section title="Links">
          <DependencyChain
            task={base}
            dependencies={dependencies}
            dependents={dependents}
            onOpen={onOpenTask}
            onUnlink={(id) =>
              link(dropDep.mutateAsync({ todoId: base.id, dependsOnId: id }))
            }
          />
          <DependencyPicker
            taskId={base.id}
            existing={dependencies}
            onPick={(id) =>
              link(addDep.mutateAsync({ todoId: base.id, dependsOnId: id }))
            }
          />
          {/* A refused link is the one case where nothing on screen changes, so
              without this the picker just clears and the user is left guessing
              whether it worked. */}
          {errors.dependsOnId && (
            <p className="mt-2 text-[12px] text-late">{errors.dependsOnId}</p>
          )}
        </Section>
      )}

      {base && (
        <Section title="History">
          <TaskHistory todoId={base.id} onOpenTask={onOpenTask} />
        </Section>
      )}

      <footer className="sticky bottom-0 mt-auto flex items-center gap-2 border-t border-rule bg-canvas px-4 py-3">
        {base && (
          // Disabled until the chain resolves. A click landing in the pending
          // window sees an empty list and deletes a blocker with no warning.
          <Button
            variant="danger"
            onClick={() => (dependents.length > 0 ? setConfirming(true) : void destroy())}
            disabled={busy || chainPending}
            data-testid="todo-delete"
          >
            Delete
          </Button>
        )}
        {base && base.status !== 'completed' && (
          <Button onClick={finish} disabled={busy || blockers.length > 0} data-testid="todo-complete">
            Complete
          </Button>
        )}
        <Button
          variant="primary"
          className="ml-auto"
          onClick={save}
          disabled={busy}
          data-testid="todo-save"
        >
          {isNew ? 'Create task' : 'Save changes'}
        </Button>
      </footer>

      {/* Only completing a task releases what depends on it, so deleting a
          blocker strands its dependents. Naming them before it happens is what
          makes the strict rule liveable. */}
      {confirming && base && (
        <ConfirmDialog
          title="Delete this task?"
          confirmLabel="Delete anyway"
          onConfirm={destroy}
          onCancel={() => setConfirming(false)}
        >
          <p className="mt-1.5 text-[13px] text-ink-soft">
            {dependents.length} task{dependents.length > 1 ? 's' : ''} depend
            {dependents.length > 1 ? '' : 's'} on this one and will stay blocked, because only
            completing a task releases what waits on it.
          </p>
          <ul className="mt-2 space-y-1">
            {dependents.map((d) => (
              <li key={d.id} className="truncate text-[13px] text-ink">
                {d.name}
              </li>
            ))}
          </ul>
        </ConfirmDialog>
      )}
    </aside>
  )
}
