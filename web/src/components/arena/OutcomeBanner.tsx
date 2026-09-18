import confetti from 'canvas-confetti'
import { motion } from 'motion/react'
import { useEffect, useRef } from 'react'
import { Handshake, Trophy } from 'lucide-react'
import type { Outcome } from '../../api/types'
import CountUp from '../text/CountUp'
import ShinyText from '../text/ShinyText'

/** Turn the engine's snake_case reason into something a spectator reads. */
const REASON_COPY: Record<string, string> = {
  solved: 'first to pass every hidden test',
  broke_and_survived: 'broke the opponent and survived the counter-attack',
  defender_solved: 'only fighter to produce a working solution',
  fewer_tokens: 'tie on every axis — won on token efficiency',
  more_cases: 'passed more hidden tests',
  forfeit: 'opponent forfeited',
  both_forfeit: 'both fighters forfeited',
  draw: 'dead even',
}

export default function OutcomeBanner({ outcome }: { outcome: Outcome }) {
  const fired = useRef(false)
  const winner = outcome.winner_id ? outcome.scores?.[outcome.winner_id] : undefined

  useEffect(() => {
    if (fired.current || !winner) return
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    fired.current = true

    // Two offset bursts read as a crowd popping, not a single firework.
    const shoot = (x: number, delay: number) =>
      window.setTimeout(
        () =>
          confetti({
            particleCount: 55,
            spread: 62,
            startVelocity: 34,
            origin: { x, y: 0.72 },
            colors: ['#f0b23f', '#3ad4cf', '#f4568f', '#eceaf0'],
            disableForReducedMotion: true,
            scalar: 0.85,
          }),
        delay,
      )
    const t1 = shoot(0.32, 90)
    const t2 = shoot(0.68, 230)
    return () => {
      clearTimeout(t1)
      clearTimeout(t2)
    }
  }, [winner])

  const copy = REASON_COPY[outcome.reason] ?? outcome.reason.replace(/_/g, ' ')

  return (
    <motion.div
      initial={{ opacity: 0, y: 14, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ type: 'spring', stiffness: 260, damping: 24 }}
      className="relative overflow-hidden rounded-xl border border-gold/45 bg-gradient-to-br from-gold/12 via-arena-surface to-arena-surface p-5"
    >
      <div className="pointer-events-none absolute -right-8 -top-10 h-32 w-32 rounded-full bg-gold/10 blur-3xl" />

      <div className="relative flex flex-wrap items-center gap-x-4 gap-y-2">
        {winner ? (
          <>
            <Trophy className="shrink-0 text-gold" size={26} />
            <div className="min-w-0">
              <div className="text-[10px] font-bold uppercase tracking-[0.22em] text-gold/70">
                Winner
              </div>
              <ShinyText
                text={winner.model}
                className="font-display text-2xl font-bold tracking-tight"
              />
            </div>
          </>
        ) : (
          <>
            <Handshake className="shrink-0 text-ink-dim" size={24} />
            <div>
              <div className="text-[10px] font-bold uppercase tracking-[0.22em] text-ink-faint">
                Result
              </div>
              <div className="font-display text-2xl font-bold tracking-tight text-ink">Draw</div>
            </div>
          </>
        )}

        <p className="w-full text-xs text-ink-dim sm:ml-auto sm:w-auto sm:max-w-xs sm:text-right">
          {copy}
        </p>
      </div>

      <div className="relative mt-4 grid grid-cols-2 gap-3 border-t border-gold/15 pt-3 sm:grid-cols-4">
        {(['A', 'B'] as const).flatMap(id => {
          const s = outcome.scores?.[id]
          if (!s) return []
          return [
            <Stat
              key={`${id}-tok`}
              label={`${id} tokens`}
              value={s.tokens_in + s.tokens_out}
              accent={id}
            />,
            <Stat
              key={`${id}-cost`}
              label={`${id} cost`}
              value={s.cost_usd}
              decimals={4}
              prefix="$"
              accent={id}
            />,
          ]
        })}
      </div>
    </motion.div>
  )
}

function Stat({
  label,
  value,
  decimals = 0,
  prefix = '',
  accent,
}: {
  label: string
  value: number
  decimals?: number
  prefix?: string
  accent: 'A' | 'B'
}) {
  return (
    <div>
      <div
        className={`font-mono text-sm font-bold tabular-nums ${
          accent === 'A' ? 'text-fighter-a' : 'text-fighter-b'
        }`}
      >
        <CountUp to={value} decimals={decimals} prefix={prefix} />
      </div>
      <div className="mt-0.5 text-[10px] uppercase tracking-wider text-ink-faint">{label}</div>
    </div>
  )
}
