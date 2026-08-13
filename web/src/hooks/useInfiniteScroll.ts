import { useEffect, useRef } from 'react'

/*
 * Loads the next page when a sentinel below the list comes into view.
 *
 * The margin fires it before the sentinel is actually visible, so the rows are
 * already there by the time you reach them. The Load more button stays: an
 * observer is invisible to a keyboard, and a page nobody can reach by tabbing
 * is a page nobody can reach.
 */
export function useInfiniteScroll(enabled: boolean, onReach: () => void) {
  const sentinel = useRef<HTMLDivElement>(null)

  // Kept in a ref so a changing callback does not tear down the observer, which
  // would fire it again on every render.
  const latest = useRef(onReach)
  latest.current = onReach

  useEffect(() => {
    const node = sentinel.current
    if (!node || !enabled) return

    const observer = new IntersectionObserver(
      ([entry]) => entry.isIntersecting && latest.current(),
      { rootMargin: '600px' },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [enabled])

  return sentinel
}
