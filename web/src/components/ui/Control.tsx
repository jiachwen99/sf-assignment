import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

/*
 * The form controls, sharing one set of dimensions so a select and an input
 * line up on the same row without anybody measuring.
 *
 * The two things that made these drift when they were inline strings: an 8px
 * height that some copies dropped, and the hover border, which is the only cue
 * that a control is interactive before you focus it.
 */

const control =
  'w-full rounded-md border border-rule-firm bg-canvas text-[13px] text-ink transition-colors placeholder:text-ink-faint hover:border-ink-faint focus:border-action focus:outline-none'

const oneLine = `h-8 px-2 ${control}`

export function Input({ className = '', ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={`${oneLine} ${className}`} {...rest} />
}

export function Select({ className = '', ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return <select className={`${oneLine} ${className}`} {...rest} />
}

export function TextArea({ className = '', ...rest }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={`resize-y px-2 py-1.5 ${control} ${className}`} {...rest} />
}

// A label above its control. Wrapping in <label> means the caption is part of
// the hit target and screen readers pair them without an id to thread through.
export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-[12px] text-ink-soft">{label}</span>
      {children}
    </label>
  )
}
