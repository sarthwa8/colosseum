import { useEffect, useRef, useState } from 'react'

/**
 * Character-scramble reveal. Adapted from the React Bits `DecryptedText`
 * effect (reactbits.dev, MIT).
 *
 * Used for model names and problem titles: an AI-vs-AI arena earns a
 * "resolving an identity" beat, and it hides the layout shift when a long
 * model id swaps in.
 *
 * Accessibility: the real text is always present for screen readers via
 * aria-label; only the visual layer scrambles.
 */

const GLYPHS = 'ABCDEFGHJKLMNPQRSTUVWXYZ0123456789#%&<>/\\[]{}*+=-'

export interface DecryptedTextProps {
  text: string
  /** ms between scramble frames */
  speed?: number
  /** how many frames each character scrambles before locking */
  perChar?: number
  className?: string
  /** replay whenever this value changes */
  trigger?: unknown
}

export default function DecryptedText({
  text,
  speed = 28,
  perChar = 3,
  className = '',
  trigger,
}: DecryptedTextProps) {
  const [shown, setShown] = useState(text)
  const raf = useRef<number | null>(null)

  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      setShown(text)
      return
    }

    let frame = 0
    const total = text.length * perChar + 6
    let timer: number

    const tick = () => {
      frame++
      const locked = Math.floor(frame / perChar)
      setShown(
        text
          .split('')
          .map((ch, i) => {
            if (i < locked || ch === ' ') return ch
            return GLYPHS[Math.floor(Math.random() * GLYPHS.length)]
          })
          .join(''),
      )
      if (frame < total) timer = window.setTimeout(tick, speed)
      else setShown(text)
    }
    timer = window.setTimeout(tick, speed)

    return () => {
      window.clearTimeout(timer)
      if (raf.current) cancelAnimationFrame(raf.current)
    }
  }, [text, speed, perChar, trigger])

  return (
    <span className={className} aria-label={text}>
      <span aria-hidden="true">{shown}</span>
    </span>
  )
}
