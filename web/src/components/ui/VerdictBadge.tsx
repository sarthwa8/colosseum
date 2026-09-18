import { motion } from 'motion/react'
import { VERDICT_LABEL, VERDICT_TONE, type Verdict } from '../../api/types'

const TONE_CLASS = {
  pass: 'border-pass/45 text-pass bg-pass/10',
  fail: 'border-fail/45 text-fail bg-fail/10',
  limit: 'border-limit/45 text-limit bg-limit/10',
} as const

/**
 * A judge verdict. Three tones, not two: TLE/MLE/OLE are resource failures,
 * not wrong answers — amber says "your logic may be fine, your complexity
 * isn't", which matters on problems built as complexity traps.
 */
export default function VerdictBadge({
  verdict,
  index = 0,
  size = 'sm',
}: {
  verdict: Verdict
  index?: number
  size?: 'sm' | 'lg'
}) {
  const tone = VERDICT_TONE[verdict]
  return (
    <motion.span
      initial={{ opacity: 0, scale: 0.7, y: 4 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{ type: 'spring', stiffness: 520, damping: 26, delay: Math.min(index * 0.05, 0.4) }}
      title={VERDICT_LABEL[verdict]}
      className={`inline-flex items-center rounded border font-mono font-bold tracking-wider ${
        TONE_CLASS[tone]
      } ${size === 'lg' ? 'px-2.5 py-1 text-xs' : 'px-1.5 py-0.5 text-[10px]'}`}
    >
      {verdict}
    </motion.span>
  )
}
