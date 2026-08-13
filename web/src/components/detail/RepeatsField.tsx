import { SCHEDULES } from '../../constants'
import { scheduleFor, scheduleKey } from '../../lib/recurrence'
import type { RecurUnit, TodoInput } from '../../types'
import { Input, Select } from '../ui/Control'

export function RepeatsField({
  form,
  onChange,
}: {
  form: TodoInput
  onChange: (patch: Partial<TodoInput>) => void
}) {
  const key = scheduleKey(form)

  return (
    <>
      <Select
        value={key}
        onChange={(e) => onChange(scheduleFor(e.currentTarget.value))}
        aria-label="Repeat unit"
        data-testid="todo-repeats"
      >
        {SCHEDULES.map((s) => (
          <option key={s.key} value={s.key}>
            {s.label}
          </option>
        ))}
      </Select>

      {key === 'custom' && (
        <div className="mt-2 flex items-center gap-1.5 text-[13px] text-ink-soft">
          every
          <Input
            type="number"
            min={2}
            value={form.recurInterval ?? 2}
            onChange={(e) =>
              onChange({ recurInterval: Math.max(2, Number(e.currentTarget.value) || 2) })
            }
            aria-label="Repeat interval"
            data-testid="todo-repeat-interval"
            className="tabular w-16"
          />
          <Select
            value={form.recurUnit ?? 'week'}
            onChange={(e) => onChange({ recurUnit: e.currentTarget.value as RecurUnit })}
            aria-label="Custom repeat unit"
            data-testid="todo-repeat-unit"
            className="w-28"
          >
            <option value="day">days</option>
            <option value="week">weeks</option>
            <option value="month">months</option>
          </Select>
        </div>
      )}

      {form.recurUnit === 'month' && (
        <p className="mt-2 text-[12px] text-ink-faint">
          A task due the 31st lands on the last day of shorter months, then returns to the 31st.
        </p>
      )}

      {form.recurUnit && !form.dueDate && (
        <p className="mt-2 text-[12px] text-ink-soft">
          A repeating task needs a due date to count from. Set one and the next occurrence opens
          when this is completed.
        </p>
      )}
    </>
  )
}
