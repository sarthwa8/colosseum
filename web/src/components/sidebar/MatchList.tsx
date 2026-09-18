import { motion } from 'motion/react'
import { Swords, Timer } from 'lucide-react'
import type { MatchSummary } from '../../api/types'

export default function MatchList({
  matches,
  selectedId,
  onSelect,
}: {
  matches: MatchSummary[]
  selectedId: string | null
  onSelect: (id: string) => void
}) {
  if (matches.length === 0) {
    return (
      <p className="px-1 py-2 text-[11px] leading-relaxed text-ink-faint">
        No matches recorded. Run{' '}
        <code className="text-ink-dim">colosseum match --problem …</code> to create one.
      </p>
    )
  }

  return (
    <div className="space-y-1.5">
      {matches.map((m, i) => {
        const active = m.id === selectedId
        const isAD = m.format === 'attack_defense'
        return (
          <motion.button
            key={m.id}
            onClick={() => onSelect(m.id)}
            initial={{ opacity: 0, x: -6 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: Math.min(i * 0.02, 0.25) }}
            className={`w-full rounded-lg border p-2.5 text-left transition-colors ${
              active
                ? 'border-gold/50 bg-gold/8'
                : 'border-arena-line bg-arena-surface/40 hover:border-arena-line hover:bg-arena-raised/50'
            }`}
          >
            <div className="flex items-center gap-1.5">
              {isAD ? (
                <Swords size={11} className="shrink-0 text-gold" />
              ) : (
                <Timer size={11} className="shrink-0 text-ink-faint" />
              )}
              <span className="min-w-0 flex-1 truncate text-xs font-medium text-ink">
                {m.problem}
              </span>
            </div>

            <div className="mt-1 truncate text-[11px] text-ink-dim">
              {(m.models ?? []).join('  vs  ')}
            </div>

            <div className="mt-1 flex items-center gap-1.5 text-[10px]">
              {m.winner ? (
                <span className="truncate text-pass">▶ {m.winner}</span>
              ) : (
                <span className="text-ink-faint">draw</span>
              )}
              <span className="ml-auto shrink-0 truncate text-ink-faint">
                {m.reason.replace(/_/g, ' ')}
              </span>
            </div>
          </motion.button>
        )
      })}
    </div>
  )
}
