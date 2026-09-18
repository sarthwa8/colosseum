import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useEffect, useRef, useState } from 'react'
import { Swords } from 'lucide-react'
import type { MatchRecord, Side } from '../../api/types'
import type { ReplayState } from '../../replay/state'
import { useReplay } from '../../replay/useReplay'
import DecryptedText from '../text/DecryptedText'
import EventFeed from './EventFeed'
import FighterCard from './FighterCard'
import OutcomeBanner from './OutcomeBanner'
import ReplayControls from './ReplayControls'

export default function Arena({ record }: { record: MatchRecord }) {
  const replay = useReplay(record)
  const { state } = replay
  const man = record.manifest

  const reduceMotion = useReducedMotion()

  // The break flash is a one-frame signal from the reducer: `breakFlash` is an
  // object only on the exact event that caused the break, and null on every
  // other. That means the effect's dependency flips back to null on the very
  // next cursor tick — so the hold timer must NOT be owned by the effect's
  // cleanup, or React cancels it before it can fire and `impact` latches on
  // forever (no second flash, and a card stuck mid-shake).
  const [impact, setImpact] = useState<{ attacker: Side; defender: Side; seq: number } | null>(null)
  const hold = useRef<number | undefined>(undefined)
  const flash = state.breakFlash

  useEffect(() => {
    if (!flash) return
    window.clearTimeout(hold.current)
    setImpact({ attacker: flash.actor, defender: flash.actor === 'A' ? 'B' : 'A', seq: flash.seq })
    hold.current = window.setTimeout(() => setImpact(null), 700)
  }, [flash])

  // Seeking away from a break should drop a held impact immediately rather than
  // leaving a card shaking at a cursor where nothing happened.
  useEffect(() => {
    if (!flash && impact) {
      window.clearTimeout(hold.current)
      hold.current = window.setTimeout(() => setImpact(null), 120)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [replay.cursor])

  useEffect(() => () => window.clearTimeout(hold.current), [])

  const winnerId = replay.atEnd || state.finished ? record.outcome?.winner_id : ''
  const isAD = man.format === 'attack_defense'

  return (
    <div className="relative flex min-h-0 flex-col gap-4">
      {/* BROKE DEFENDER impact wash. Keyed on seq so a second break remounts
          and re-animates instead of silently reusing the first one's element.
          Suppressed entirely under reduced motion — a full-viewport flash is
          exactly what that setting exists to prevent. */}
      <AnimatePresence>
        {impact && !reduceMotion && (
          <motion.div
            key={impact.seq}
            initial={{ opacity: 0 }}
            animate={{ opacity: [0, 0.85, 0] }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.65, times: [0, 0.18, 1] }}
            className="pointer-events-none fixed inset-0 z-40"
            style={{
              background:
                'radial-gradient(ellipse at center, rgba(74,222,128,0.16), transparent 62%)',
            }}
          >
            <motion.div
              initial={{ scale: 0.82, opacity: 0 }}
              animate={{ scale: [0.82, 1.04, 1], opacity: [0, 1, 0] }}
              transition={{ duration: 0.7, times: [0, 0.25, 1] }}
              className="absolute inset-0 grid place-items-center"
            >
              <div className="flex items-center gap-3 rounded-2xl border border-pass/50 bg-arena-void/85 px-7 py-4 backdrop-blur-sm">
                <Swords className="text-pass" size={26} />
                <span className="font-display text-2xl font-bold tracking-[0.12em] text-pass">
                  BROKE DEFENDER
                </span>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>

      <header className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <h1 className="font-display text-xl font-bold tracking-tight text-ink">
          <DecryptedText text={man.problem} trigger={man.match_id} />
        </h1>
        <span
          className={`rounded border px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider ${
            isAD ? 'border-gold/40 bg-gold/10 text-gold' : 'border-arena-line text-ink-dim'
          }`}
        >
          {isAD ? 'attack / defense' : 'race'}
        </span>
        {state.phase && (
          <span className="text-[11px] text-ink-faint">phase: {state.phase}</span>
        )}
        <span className="ml-auto font-mono text-[11px] text-ink-faint">{man.match_id}</span>
      </header>

      <div className="grid gap-4 lg:grid-cols-[1fr_auto_1fr]">
        <FighterCard
          fighter={state.fighters.A}
          winner={winnerId === 'A'}
          loser={!!winnerId && winnerId !== 'A'}
          shaking={!reduceMotion && impact?.defender === 'A'}
        />

        <div className="hidden items-center justify-center lg:flex">
          <motion.div
            animate={impact && !reduceMotion ? { scale: [1, 1.35, 1], rotate: [0, -12, 0] } : {}}
            transition={{ duration: 0.5 }}
            className="font-display text-sm font-bold tracking-widest text-ink-faint"
          >
            VS
          </motion.div>
        </div>

        <FighterCard
          fighter={state.fighters.B}
          winner={winnerId === 'B'}
          loser={!!winnerId && winnerId !== 'B'}
          shaking={!reduceMotion && impact?.defender === 'B'}
        />
      </div>

      <ReplayControls replay={replay} events={record.events ?? []} />

      {/* keyed so AnimatePresence can track it: scrubbing back off the end
          fades the banner out instead of popping it. */}
      <AnimatePresence>
        {(replay.atEnd || state.finished) && record.outcome && (
          <OutcomeBanner key="outcome" outcome={record.outcome} />
        )}
      </AnimatePresence>

      {/* flex-1 can't grow inside an auto-height column, so the log was pinned
          at its min-height forever. Give it a viewport-relative height instead. */}
      <div className="h-[clamp(220px,34vh,460px)]">
        <EventFeed feed={state.feed} />
      </div>

      {/* Screen readers get the beats, not all 25 events — announcing every
          tick of a 4x autoplay is noise, not information. */}
      <p aria-live="polite" className="sr-only">
        {liveSummary(state, replay.atEnd, record.outcome?.reason)}
      </p>
    </div>
  )
}

/**
 * A one-line spoken summary of where the replay is. Deliberately coarse: the
 * feed itself is a `role="log"`, and announcing every event of a 4x autoplay
 * would bury the moments that matter.
 */
function liveSummary(state: ReplayState, atEnd: boolean, reason?: string): string {
  if (atEnd || state.finished) return `Match over. ${reason?.replace(/_/g, ' ') ?? ''}`.trim()
  const broke = (['A', 'B'] as const).find(s => state.fighters[s].broke)
  if (broke) return `Fighter ${broke} broke the opponent's solution.`
  if (state.phase) return `${state.phase} phase.`
  return ''
}
