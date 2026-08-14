import { useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'

import { queryKeys } from '../constants'

/*
 * Listens for changes anybody makes and invalidates, rather than patching rows
 * out of the event payload.
 *
 * The payload names the task and what happened to it and nothing more. Applying
 * a payload would make the stream a second source of truth that can disagree
 * with the first, and would leave a dropped or out-of-order event permanently
 * wrong. Invalidating is self-correcting: the next event repairs whatever the
 * last one missed.
 *
 * The keys are hierarchical, so invalidating ['todos'] reaches every list,
 * chain and history under it in one call.
 */
export function useEvents() {
  const qc = useQueryClient()

  useEffect(() => {
    const source = new EventSource('/api/events')

    const refresh = () => {
      qc.invalidateQueries({ queryKey: queryKeys.todos })
      qc.invalidateQueries({ queryKey: queryKeys.counts })
      qc.invalidateQueries({ queryKey: queryKeys.trash })
    }

    source.addEventListener('change', refresh)

    // EventSource reconnects on its own, so an error is not a reason to close
    // it. Closing here would turn a dropped connection into a permanently dead
    // stream.
    return () => source.close()
  }, [qc])
}
