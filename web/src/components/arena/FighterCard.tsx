import { AnimatePresence, motion } from 'motion/react'
import { useState } from 'react'
import { ChevronRight, Code2, ShieldCheck, Swords, XOctagon } from 'lucide-react'
import type { Side } from '../../api/types'
import type { FighterState } from '../../replay/state'
import DecryptedText from '../text/DecryptedText'
import TestMeter from '../ui/TestMeter'
import VerdictBadge from '../ui/VerdictBadge'

const SKIN = {
  A: {
    text: 'text-fighter-a',
    border: 'border-fighter-a/35',
    glow: 'shadow-[0_0_38px_-14px_var(--color-fighter-a)]',
    chip: 'bg-fighter-a/12 text-fighter-a border-fighter-a/30',
    rail: 'from-fighter-a/70',
  },
  B: {
    text: 'text-fighter-b',
    border: 'border-fighter-b/35',
    glow: 'shadow-[0_0_38px_-14px_var(--color-fighter-b)]',
    chip: 'bg-fighter-b/12 text-fighter-b border-fighter-b/30',
    rail: 'from-fighter-b/70',
  },
} as const satisfies Record<Side, Record<string, string>>

export default function FighterCard({
  fighter,
  winner,
  loser,
  shaking,
}: {
  fighter: FighterState
  winner: boolean
  loser: boolean
  /** true on the frame this fighter's code was broken */
  shaking: boolean
}) {
  const [showCode, setShowCode] = useState(false)
  const skin = SKIN[fighter.side]
  const thinking = fighter.phase === 'thinking'
  const failedLast =
    fighter.verdicts.length > 0 && fighter.verdicts[fighter.verdicts.length - 1] !== 'AC'

  return (
    <motion.section
      animate={
        shaking
          ? { x: [0, -9, 8, -6, 4, 0], rotate: [0, -0.5, 0.45, -0.3, 0] }
          : { x: 0, rotate: 0 }
      }
      transition={{ duration: 0.42 }}
      className={`relative overflow-hidden rounded-xl border bg-arena-surface/80 backdrop-blur-sm transition-colors duration-500 ${
        winner ? `border-gold/55 ${skin.glow}` : loser ? 'border-arena-line opacity-70' : skin.border
      }`}
    >
      {/* Side rail: the fastest "who am I looking at" cue on the card. */}
      <div className={`absolute inset-y-0 left-0 w-[3px] bg-gradient-to-b ${skin.rail} to-transparent`} />

      {/* Thinking sweep — motion that means "work is happening", not decoration. */}
      {thinking && (
        <div className="pointer-events-none absolute inset-x-0 top-0 h-px overflow-hidden">
          <div className={`h-px w-1/3 animate-sweep bg-gradient-to-r from-transparent via-current to-transparent ${skin.text}`} />
        </div>
      )}

      <div className="p-4">
        <header className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className={`font-display text-2xl font-bold leading-none ${skin.text}`}>
                {fighter.side}
              </span>
              {winner && (
                <motion.span
                  initial={{ opacity: 0, scale: 0.6 }}
                  animate={{ opacity: 1, scale: 1 }}
                  className="rounded border border-gold/45 bg-gold/12 px-1.5 py-0.5 text-[10px] font-bold tracking-widest text-gold"
                >
                  WINNER
                </motion.span>
              )}
            </div>
            <DecryptedText
              text={fighter.model}
              trigger={fighter.model}
              className="mt-1 block truncate text-sm font-medium text-ink"
            />
          </div>

          <div className="shrink-0 text-right">
            <div className="font-mono text-lg font-bold tabular-nums text-ink">
              {fighter.passed}
              <span className="text-ink-faint">/{fighter.total || '—'}</span>
            </div>
            <div className="text-[10px] uppercase tracking-wider text-ink-faint">tests</div>
          </div>
        </header>

        <div className="mt-3">
          <TestMeter
            passed={fighter.passed}
            total={fighter.total}
            side={fighter.side}
            failed={failedLast}
          />
        </div>

        <div className="mt-3 flex min-h-[18px] items-center gap-2 text-xs text-ink-dim">
          {thinking && (
            <span className="relative flex h-1.5 w-1.5 shrink-0">
              <span className={`absolute inline-flex h-full w-full animate-pulse-ring rounded-full ${skin.text} bg-current`} />
              <span className={`relative inline-flex h-1.5 w-1.5 rounded-full ${skin.text} bg-current`} />
            </span>
          )}
          <span className="truncate">{fighter.status}</span>
        </div>

        {fighter.verdicts.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1">
            {fighter.verdicts.map((v, i) => (
              <VerdictBadge key={i} verdict={v} index={i} />
            ))}
          </div>
        )}

        <AnimatePresence>
          {(fighter.broke || fighter.shielded || fighter.forfeit) && (
            <motion.div
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              className="mt-3 flex flex-wrap gap-1.5"
            >
              {fighter.broke && (
                <span className="inline-flex items-center gap-1 rounded border border-pass/40 bg-pass/10 px-2 py-0.5 text-[10px] font-bold tracking-wide text-pass">
                  <Swords size={11} /> BROKE DEFENDER
                </span>
              )}
              {fighter.shielded && (
                <span className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-[10px] font-bold tracking-wide ${skin.chip}`}>
                  <ShieldCheck size={11} /> HELD
                </span>
              )}
              {fighter.forfeit && (
                <span className="inline-flex items-center gap-1 rounded border border-fail/40 bg-fail/10 px-2 py-0.5 text-[10px] font-bold tracking-wide text-fail">
                  <XOctagon size={11} /> FORFEIT
                </span>
              )}
            </motion.div>
          )}
        </AnimatePresence>

        {fighter.code && (
          <div className="mt-3">
            <button
              onClick={() => setShowCode(s => !s)}
              className="flex w-full items-center gap-1.5 rounded border border-arena-line bg-arena-raised/60 px-2 py-1.5 text-[11px] text-ink-dim transition-colors hover:border-arena-line hover:text-ink"
            >
              <Code2 size={12} />
              <span>solution</span>
              <ChevronRight
                size={12}
                className={`ml-auto transition-transform ${showCode ? 'rotate-90' : ''}`}
              />
            </button>
            <AnimatePresence initial={false}>
              {showCode && (
                <motion.pre
                  initial={{ height: 0, opacity: 0 }}
                  animate={{ height: 'auto', opacity: 1 }}
                  exit={{ height: 0, opacity: 0 }}
                  transition={{ duration: 0.2 }}
                  className="mt-1.5 max-h-56 overflow-auto rounded border border-arena-line bg-arena-void/80 p-2.5 text-[11px] leading-relaxed text-ink-dim"
                >
                  <code>{fighter.code}</code>
                </motion.pre>
              )}
            </AnimatePresence>
          </div>
        )}
      </div>
    </motion.section>
  )
}
