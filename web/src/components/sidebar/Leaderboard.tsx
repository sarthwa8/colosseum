import { motion } from 'motion/react'
import type { Report } from '../../api/types'
import CountUp from '../text/CountUp'

/**
 * Standings. Two separate rate bars on purpose: solve-rate is the axis that
 * saturates, break-rate is the axis that separates — collapsing them into one
 * score would undo the entire point of the project.
 *
 * The Elo confidence interval is drawn as a band rather than printed as "±44",
 * so a rating computed from a handful of games visibly reads as noise.
 */
export default function Leaderboard({ report }: { report: Report | null }) {
  const models = report?.models ?? []
  if (models.length === 0) {
    return (
      <p className="px-1 py-2 text-[11px] leading-relaxed text-ink-faint">
        No ladder report yet. Run <code className="text-ink-dim">colosseum ladder</code> to
        populate standings.
      </p>
    )
  }

  const elos = models
    .map(m => report?.robustness_elo?.[m.model])
    .filter((e): e is NonNullable<typeof e> => !!e)
  const lo = Math.min(...elos.map(e => e.low), 1450)
  const hi = Math.max(...elos.map(e => e.high), 1550)
  const span = Math.max(hi - lo, 1)

  return (
    <div className="space-y-2">
      {models.map((m, i) => {
        const elo = report?.robustness_elo?.[m.model]
        return (
          <motion.div
            key={m.model}
            initial={{ opacity: 0, x: -6 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: Math.min(i * 0.05, 0.3) }}
            className="rounded-lg border border-arena-line bg-arena-surface/50 p-2.5"
          >
            <div className="flex items-baseline gap-2">
              <span className="font-mono text-[10px] text-ink-faint">{i + 1}</span>
              <span className="min-w-0 flex-1 truncate text-xs font-medium text-ink">
                {m.model}
              </span>
              {elo && (
                <span className="font-mono text-[11px] font-bold tabular-nums text-gold">
                  <CountUp to={elo.rating} />
                </span>
              )}
            </div>

            {elo && (
              /* Elo with its bootstrap CI drawn to scale. A wide band means
                 "not enough games to trust this gap". */
              <div className="relative mt-2 h-1 rounded-full bg-arena-line/50">
                <div
                  className="absolute h-1 rounded-full bg-gold/25"
                  style={{
                    left: `${((elo.low - lo) / span) * 100}%`,
                    width: `${((elo.high - elo.low) / span) * 100}%`,
                  }}
                />
                <div
                  className="absolute top-1/2 h-2 w-0.5 -translate-y-1/2 rounded-full bg-gold"
                  style={{ left: `${((elo.rating - lo) / span) * 100}%` }}
                />
              </div>
            )}

            <div className="mt-2 grid grid-cols-2 gap-2">
              <Rate label="solve" value={m.solve_rate} tone="var(--color-fighter-a)" />
              <Rate label="break" value={m.break_rate} tone="var(--color-pass)" />
            </div>

            <div className="mt-1.5 flex justify-between font-mono text-[10px] text-ink-faint">
              <span>{m.games} games</span>
              {m.cost_usd > 0 && <span>${m.cost_usd.toFixed(2)}</span>}
            </div>
          </motion.div>
        )
      })}
    </div>
  )
}

function Rate({ label, value, tone }: { label: string; value: number; tone: string }) {
  return (
    <div>
      <div className="flex justify-between text-[10px] text-ink-faint">
        <span>{label}</span>
        <span className="font-mono tabular-nums">{Math.round(value * 100)}%</span>
      </div>
      <div className="mt-0.5 h-1 rounded-full bg-arena-line/50">
        <motion.div
          className="h-1 rounded-full"
          style={{ background: tone }}
          initial={{ width: 0 }}
          animate={{ width: `${value * 100}%` }}
          transition={{ type: 'spring', stiffness: 120, damping: 22 }}
        />
      </div>
    </div>
  )
}
