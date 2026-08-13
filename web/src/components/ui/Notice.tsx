import type { ReactNode } from 'react'

// A boxed message with something to do about it. Amber because the thing it
// reports is usually a wait rather than a failure.
export function Notice({
  title,
  children,
  className = '',
  ...rest
}: { title: string; children?: ReactNode } & React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={`rounded-md border border-halt-edge bg-halt-wash p-3 ${className}`} {...rest}>
      <p className="text-[13px] font-medium text-ink">{title}</p>
      {children}
    </div>
  )
}

export function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="border-t border-rule px-4 py-3.5">
      <h3 className="mb-2.5 text-[11px] font-medium tracking-wide text-ink-soft uppercase">
        {title}
      </h3>
      {children}
    </section>
  )
}
