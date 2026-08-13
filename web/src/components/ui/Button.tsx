import type { ButtonHTMLAttributes } from 'react'

/*
 * Every button in the app, in four intents and two sizes.
 *
 * Before this there were five bordered-button class strings that differed only
 * in whether they carried `bg-canvas`, `font-medium` or `disabled:opacity-60`
 * against `-50`. Nobody chose those differences; they accumulated.
 */

type Variant = 'primary' | 'outline' | 'quiet' | 'danger' | 'destructive' | 'contrast'
type Size = 'sm' | 'md'

const base =
  'inline-flex items-center justify-center rounded-md font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60'

const sizes: Record<Size, string> = {
  sm: 'px-2.5 py-1 text-[12px]',
  md: 'px-3 py-1.5 text-[13px]',
}

const variants: Record<Variant, string> = {
  primary: 'bg-action text-white hover:bg-action-hover',
  outline: 'border border-rule-firm bg-canvas text-ink hover:bg-sunk',
  quiet: 'font-normal text-ink-soft hover:bg-sunk hover:text-ink',
  // Destructive intent shows on hover rather than at rest, so a row of actions
  // does not read as a warning until you reach for the one that is.
  danger: 'font-normal text-ink-soft hover:bg-late-wash hover:text-late',
  // Filled red, for the confirmation you have already been warned about.
  destructive: 'bg-late text-white hover:opacity-90',
  contrast: 'bg-ink text-canvas hover:bg-ink-soft',
}

export function Button({
  variant = 'outline',
  size = 'md',
  full = false,
  className = '',
  type = 'button',
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant
  size?: Size
  full?: boolean
}) {
  return (
    <button
      type={type}
      className={`${base} ${sizes[size]} ${variants[variant]} ${full ? 'w-full' : ''} ${className}`}
      {...rest}
    />
  )
}

// The close cross, the only icon-shaped button in the app.
export function IconButton({
  className = '',
  type = 'button',
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      type={type}
      className={`grid size-7 shrink-0 place-items-center rounded-md text-ink-faint transition-colors hover:bg-sunk hover:text-ink ${className}`}
      {...rest}
    />
  )
}
