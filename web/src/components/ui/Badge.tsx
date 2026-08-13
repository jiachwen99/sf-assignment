import type { ReactNode } from 'react'

// The small outlined pill: a repeat cadence, a blocked count. Always carries a
// word, never a bare glyph, so a reader is not made to hover to find out.
const tones = {
  neutral: 'border-rule-firm text-ink-soft',
  halt: 'border-halt-edge bg-halt-wash text-halt',
}

export function Badge({
  children,
  tone = 'neutral',
  className = '',
  ...rest
}: {
  children: ReactNode
  tone?: keyof typeof tones
} & React.HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={`inline-flex shrink-0 items-center gap-1 rounded border px-1.5 py-px text-[11px] ${tones[tone]} ${className}`}
      {...rest}
    >
      {children}
    </span>
  )
}
