import { useEffect, useRef, useState } from 'react'

/**
 * Animated numeric roll-up. Adapted from the React Bits `CountUp` effect
 * (reactbits.dev, MIT), with an eased curve so the number decelerates into
 * its final value instead of stopping dead.
 */
export interface CountUpProps {
  to: number
  from?: number
  duration?: number
  decimals?: number
  prefix?: string
  suffix?: string
  className?: string
}

const easeOutExpo = (t: number) => (t === 1 ? 1 : 1 - Math.pow(2, -10 * t))

export default function CountUp({
  to,
  from = 0,
  duration = 900,
  decimals = 0,
  prefix = '',
  suffix = '',
  className = '',
}: CountUpProps) {
  const [value, setValue] = useState(from)
  const raf = useRef<number | null>(null)

  useEffect(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      setValue(to)
      return
    }
    const start = performance.now()
    const run = (now: number) => {
      const t = Math.min(1, (now - start) / duration)
      setValue(from + (to - from) * easeOutExpo(t))
      if (t < 1) raf.current = requestAnimationFrame(run)
    }
    raf.current = requestAnimationFrame(run)
    return () => {
      if (raf.current) cancelAnimationFrame(raf.current)
    }
  }, [to, from, duration])

  return (
    <span className={className}>
      {prefix}
      {value.toFixed(decimals)}
      {suffix}
    </span>
  )
}
