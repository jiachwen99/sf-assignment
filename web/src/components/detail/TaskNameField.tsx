import type { TodoInput } from '../../types'
import { Field, TextArea } from '../ui/Control'

// A textarea rather than an input, because an input cannot wrap and a long name
// is then cut off mid-word.
export function TaskNameField({
  form,
  onChange,
  error,
  autoFocus = false,
}: {
  form: TodoInput
  onChange: (name: string) => void
  error?: string
  autoFocus?: boolean
}) {
  return (
    <div>
      <Field label="Name">
        <TextArea
          rows={2}
          autoFocus={autoFocus}
          value={form.name}
          onChange={(e) => onChange(e.currentTarget.value)}
          onKeyDown={(e) => e.key === 'Enter' && e.preventDefault()}
          placeholder="What needs doing?"
          aria-label="Task name"
          data-testid="todo-name"
        />
      </Field>
      {error && <p className="mt-1 text-[12px] text-late">{error}</p>}
    </div>
  )
}
