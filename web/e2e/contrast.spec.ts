import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

import { expect, test } from '@playwright/test'

/*
 * The palette is written in OKLCH, which is chosen for even lightness steps and
 * says nothing on its own about whether text can be read on the surface behind
 * it. This converts the tokens the same way a browser does and holds the pairs
 * that actually occur in the interface to WCAG AA.
 *
 * It reads styles.css rather than a copy of the values, because a table of
 * numbers written beside the palette is a claim that goes stale the first time
 * somebody nudges a lightness and does not think to update it. Anything read
 * from the source cannot drift from the source.
 *
 * No browser: this is arithmetic over a file. It lives here because this is the
 * only test project the front end has, not because it needs a page.
 */

const css = readFileSync(fileURLToPath(new URL('../src/styles.css', import.meta.url)), 'utf8')

type Oklch = [number, number, number]

function tokens(): Record<string, Oklch> {
  const found: Record<string, Oklch> = {}
  // --color-name: oklch(L C H);  chroma and hue are optional in the source.
  const pattern = /--color-([\w-]+):\s*oklch\(([\d.]+)\s+([\d.]+)\s+([\d.]+)\)/g
  for (const [, name, l, c, h] of css.matchAll(pattern)) {
    found[name] = [Number(l), Number(c), Number(h)]
  }
  return found
}

// OKLCH to linear sRGB, the same transform the browser applies before painting.
function linear([L, C, H]: Oklch): [number, number, number] {
  const hr = (H * Math.PI) / 180
  const a = C * Math.cos(hr)
  const b = C * Math.sin(hr)

  const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3
  const m = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3
  const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3

  return [
    4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s,
    -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s,
    -0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s,
  ].map((v) => Math.min(1, Math.max(0, v))) as [number, number, number]
}

function luminance(c: Oklch): number {
  const [r, g, b] = linear(c)
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

function ratio(fg: Oklch, bg: Oklch): number {
  const a = luminance(fg)
  const b = luminance(bg)
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
}

// Every pair where the interface puts text of one token on a surface of
// another. Read off the components rather than invented: `ink-faint` on `sunk`
// is the skeleton, `halt` on `halt-wash` is the blocked badge, and so on.
const textPairs: Array<[string, string]> = [
  ['ink', 'canvas'],
  ['ink-soft', 'canvas'],
  ['ink-faint', 'canvas'],
  ['ink-faint', 'raised'],
  ['ink-faint', 'sunk'],
  ['action', 'canvas'],
  ['action', 'action-wash'],
  ['late', 'canvas'],
  ['late', 'late-wash'],
  ['done', 'canvas'],
  ['halt', 'canvas'],
  ['halt', 'halt-wash'],
  ['canvas', 'action'],
  ['canvas', 'ink'],
]

test.describe('colour contrast', () => {
  test('every text pair in the interface clears WCAG AA', () => {
    const t = tokens()
    const failures: string[] = []

    for (const [fg, bg] of textPairs) {
      expect(t[fg], `--color-${fg} is missing from styles.css`).toBeDefined()
      expect(t[bg], `--color-${bg} is missing from styles.css`).toBeDefined()

      const r = ratio(t[fg], t[bg])
      if (r < 4.5) failures.push(`${fg} on ${bg}: ${r.toFixed(2)}:1`)
    }

    expect(failures, 'body text must reach 4.5:1').toEqual([])
  })

  /*
   * The sentinel. Without it the test above passes just as happily on an empty
   * palette or a regex that stopped matching, and a green check would mean the
   * parser broke rather than the colours are sound. This pair is deliberately
   * unreadable and must be caught.
   */
  test('the check can actually fail', () => {
    const t = tokens()
    expect(Object.keys(t).length, 'the palette should have parsed').toBeGreaterThan(10)
    expect(ratio(t['rule'], t['canvas'])).toBeLessThan(3)
  })
})
