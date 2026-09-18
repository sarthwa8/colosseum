import { motion } from 'motion/react'
import type { Side } from '../../api/types'

const ACCENT: Record<Side, string> = {
  A: 'var(--color-fighter-a)',
  B: 'var(--color-fighter-b)',
}

/**
 * Hidden-test progress, drawn like a fighting-game health bar.
 *
 * Segmented at <= 14 tests, continuous above: during a replay the question is
 * "how many of the nine passed", and counting lit cells answers it instantly
 * where a percentage bar does not. Past ~14 the cells get too thin to count,
 * so a solid fill reads better.
 */
export default function TestMeter({
  passed,
  total,
  side,
  failed = false,
}: {
  passed: number
  total: number
  side: Side
  failed?: boolean
}) {
  const color = failed ? 'var(--color-fail)' : ACCENT[side]
  const pct = total > 0 ? passed / total : 0

  if (total === 0) {
    return <div className="h-1.5 w-full rounded-full bg-arena-line/60" aria-hidden="true" />
  }

  const segmented = total <= 14

  return (
    <div
      className="w-full"
      role="progressbar"
      aria-valuenow={passed}
      aria-valuemin={0}
      aria-valuemax={total}
      aria-label={`${passed} of ${total} hidden tests passing`}
    >
      {segmented ? (
        <div className="flex gap-[3px]">
          {Array.from({ length: total }, (_, i) => {
            const lit = i < passed
            return (
              <motion.div
                key={i}
                className="h-1.5 flex-1 rounded-[2px]"
                initial={false}
                animate={{
                  backgroundColor: lit ? color : 'rgba(38,38,48,0.85)',
                  boxShadow: lit ? `0 0 7px -1px ${color}` : '0 0 0 0 transparent',
                }}
                transition={{ duration: 0.22, delay: lit ? Math.min(i * 0.035, 0.45) : 0 }}
              />
            )
          })}
        </div>
      ) : (
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-arena-line/60">
          <motion.div
            className="h-full rounded-full"
            style={{ backgroundColor: color, boxShadow: `0 0 8px -1px ${color}` }}
            initial={false}
            animate={{ width: `${pct * 100}%` }}
            transition={{ type: 'spring', stiffness: 160, damping: 26 }}
          />
        </div>
      )}
    </div>
  )
}
