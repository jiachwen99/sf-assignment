export function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <div className="grid place-items-center px-6 py-20 text-center">
      <p className="text-[15px] text-ink">No tasks yet</p>
      <p className="mt-1 max-w-sm text-[13px] text-ink-soft">
        Tasks you add appear here, newest first, with their due date, status and priority.
      </p>
      <button
        type="button"
        onClick={onCreate}
        className="mt-4 rounded-md bg-action px-3 py-1.5 text-[13px] font-medium text-white transition-colors hover:bg-action-hover"
      >
        Add a task
      </button>
    </div>
  )
}
