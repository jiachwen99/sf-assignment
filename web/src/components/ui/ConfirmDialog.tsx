import type { ReactNode } from 'react'
import { Button } from './Button'

// A modal only for the decisions that cannot be taken back. Anything that can
// be undone should just happen.
export function ConfirmDialog({
  title,
  confirmLabel,
  onConfirm,
  onCancel,
  children,
}: {
  title: string
  confirmLabel: string
  onConfirm: () => void
  onCancel: () => void
  children: ReactNode
}) {
  return (
    <div className="fixed inset-0 z-40 grid place-items-center bg-ink/25 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="w-full max-w-sm rounded-lg border border-rule bg-canvas p-4 shadow-xl"
      >
        <h2 className="text-[14px] font-medium text-ink">{title}</h2>
        {children}
        <div className="mt-4 flex justify-end gap-2">
          <Button onClick={onCancel}>Cancel</Button>
          <Button variant="destructive" onClick={onConfirm} data-testid="confirm-proceed">
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  )
}
