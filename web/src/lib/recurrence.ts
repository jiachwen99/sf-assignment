import { SCHEDULES } from '../constants'
import type { Todo, TodoInput } from '../types'

type Schedule = Pick<Todo, 'recurUnit' | 'recurInterval'>

const plural = { day: 'days', week: 'weeks', month: 'months' } as const
const named = { day: 'Daily', week: 'Weekly', month: 'Monthly' } as const

// Every one is a name, every three is arithmetic. The brief's own vocabulary is
// the common intervals, so those read as words and the rest reads as a rule.
export function recurrenceLabel(schedule: Schedule) {
  if (!schedule.recurUnit) return null
  const every = schedule.recurInterval ?? 1
  return every === 1 ? named[schedule.recurUnit] : `Every ${every} ${plural[schedule.recurUnit]}`
}

// Which entry of SCHEDULES a task's stored schedule corresponds to. Anything
// with an interval above one is Custom, whatever its unit.
export function scheduleKey(schedule: Schedule) {
  if (!schedule.recurUnit) return 'none'
  return (schedule.recurInterval ?? 1) === 1 ? schedule.recurUnit : 'custom'
}

export function scheduleFor(key: string): Pick<TodoInput, 'recurUnit' | 'recurInterval'> {
  const picked = SCHEDULES.find((s) => s.key === key) ?? SCHEDULES[0]
  return { recurUnit: picked.unit, recurInterval: picked.interval }
}
