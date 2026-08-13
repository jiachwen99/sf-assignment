/*
 * The two glyphs the app draws itself.
 *
 * The repeat arrows were pasted into the task row and the detail panel
 * separately, which is how a shape ends up meaning two slightly different
 * things. There is one of each now.
 */

export function RepeatIcon({ className = 'size-2.5' }: { className?: string }) {
  return (
    <svg viewBox="0 0 12 12" className={className} fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden>
      <path d="M1.5 6a4.5 4.5 0 0 1 7.7-3.2M10.5 6a4.5 4.5 0 0 1-7.7 3.2" strokeLinecap="round" />
      <path d="M9.2 1v1.8H7.4M2.8 11V9.2h1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export function CloseIcon({ className = 'size-3.5' }: { className?: string }) {
  return (
    <svg viewBox="0 0 14 14" className={className} fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden>
      <path d="m3 3 8 8M11 3l-8 8" strokeLinecap="round" />
    </svg>
  )
}
