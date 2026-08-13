import { recurrenceLabel } from '../../lib/recurrence'
import type { TodoInput } from '../../types'
import { Badge } from '../ui/Badge'
import { RepeatIcon } from '../ui/icons'

// A textarea rather than an input, because an input cannot wrap and a long name
// is then cut off mid-word.
export function TaskNameField({
  form,
  onChange,
}: {
  form: TodoInput
  onChange: (name: string) => void
}) {
  const repeats = recurrenceLabel(form)

  return (
    <div className="min-w-0 flex-1">
      <textarea
        rows={1}
        value={form.name}
        onChange={(e) => onChange(e.currentTarget.value)}
        onKeyDown={(e) => e.key === 'Enter' && e.preventDefault()}
        placeholder="Task name"
        aria-label="Task name"
        data-testid="todo-name"
        className="block w-full resize-none border-0 bg-transparent p-0 text-[15px] leading-snug font-medium text-ink outline-none placeholder:text-ink-faint"
      />
      {repeats && (
        <Badge data-testid="detail-recurs" className="mt-1">
          <RepeatIcon />
          {repeats}
        </Badge>
      )}
    </div>
  )
}
