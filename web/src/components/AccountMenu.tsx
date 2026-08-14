import { useEffect, useRef, useState } from 'react'

import { ApiError } from '../api/client'
import { useCurrentUser, useLogin, useLogout, useRegister } from '../api/auth'
import { Button } from './ui/Button'
import { Field, Input } from './ui/Control'

/*
 * Signing in is optional here: everyone shares one list, and an account only
 * records who changed what. So this is a quiet control in the corner rather
 * than a gate in front of the application, and the panel says as much.
 */
export function AccountMenu() {
  const { data: user } = useCurrentUser()
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [errors, setErrors] = useState<Record<string, string>>({})

  const login = useLogin()
  const register = useRegister()
  const logout = useLogout()

  const panel = useRef<HTMLDivElement>(null)

  // Closes on a click elsewhere and on Escape, because a panel that only closes
  // by its own button is a panel people leave open.
  useEffect(() => {
    if (!open) return

    const onDown = (e: MouseEvent) => {
      if (!panel.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)

    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const submit = async (form: HTMLFormElement) => {
    const data = new FormData(form)
    const email = String(data.get('email') ?? '')
    const password = String(data.get('password') ?? '')
    const name = String(data.get('name') ?? '')

    setErrors({})
    try {
      if (mode === 'register') await register.mutateAsync({ email, name, password })
      else await login.mutateAsync({ email, password })
      setOpen(false)
    } catch (err) {
      if (err instanceof ApiError && err.fields) return setErrors(err.fields)
      throw err
    }
  }

  if (user) {
    return (
      <div className="flex items-center gap-2">
        <span className="text-[12px] text-ink-soft" data-testid="current-user">
          {user.name}
        </span>
        <Button variant="quiet" size="sm" onClick={() => logout.mutate()} data-testid="logout">
          Sign out
        </Button>
      </div>
    )
  }

  return (
    <div className="relative" ref={panel}>
      <Button
        variant="quiet"
        size="sm"
        onClick={() => {
          // Reopening always starts at sign in. Leaving it on whichever tab was
          // used last means the fields under the cursor are not the ones a
          // returning user wants.
          setMode('login')
          setErrors({})
          setOpen((o) => !o)
        }}
        data-testid="open-login"
      >
        Sign in
      </Button>

      {open && (
        <div className="absolute right-0 z-30 mt-1 w-64 rounded-lg border border-rule bg-canvas p-3 shadow-lg">
          <div className="mb-2.5 flex gap-1 rounded-md bg-sunk p-0.5">
            {(['login', 'register'] as const).map((m) => (
              <button
                key={m}
                type="button"
                onClick={() => {
                  setMode(m)
                  setErrors({})
                }}
                data-testid={`tab-${m}`}
                className={`flex-1 rounded px-2 py-1 text-[12px] transition-colors ${
                  mode === m ? 'bg-canvas font-medium text-ink shadow-sm' : 'text-ink-soft'
                }`}
              >
                {m === 'login' ? 'Sign in' : 'Create account'}
              </button>
            ))}
          </div>

          <form
            className="space-y-2"
            onSubmit={(e) => {
              e.preventDefault()
              void submit(e.currentTarget)
            }}
          >
            {mode === 'register' && (
              <Field label="Name">
                <Input name="name" autoComplete="name" data-testid="register-name" />
              </Field>
            )}
            <Field label="Email">
              <Input name="email" type="email" autoComplete="email" data-testid="auth-email" />
            </Field>
            <Field label="Password">
              <Input
                name="password"
                type="password"
                autoComplete={mode === 'register' ? 'new-password' : 'current-password'}
                data-testid="auth-password"
              />
            </Field>

            {Object.values(errors).map((message) => (
              <p key={message} className="text-[12px] text-late">
                {message}
              </p>
            ))}

            <Button
              variant="primary"
              full
              type="submit"
              disabled={login.isPending || register.isPending}
              data-testid={mode === 'register' ? 'register-submit' : 'login-submit'}
            >
              {mode === 'login' ? 'Sign in' : 'Create account'}
            </Button>

            <p className="text-[11px] text-ink-faint">
              Everyone shares one list. An account only records who changed what.
            </p>
          </form>
        </div>
      )}
    </div>
  )
}
