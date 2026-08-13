import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import './styles.css'

// The walking skeleton. The task list replaces this in SF-002; it is here so
// the stack can be proved end to end before there is anything to show.
function App() {
  return (
    <main className="grid min-h-screen place-items-center bg-canvas text-ink">
      <p className="text-[13px] text-ink-soft">TODO</p>
    </main>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
