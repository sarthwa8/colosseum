import { AnimatePresence, motion } from 'motion/react'
import { useEffect, useState } from 'react'
import { Swords } from 'lucide-react'
import type { MatchRecord, Side } from '../../api/types'
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

  // The break flash is a one-frame signal from the reducer; hold it long enough
  // to see, then clear. breakFlash is a fresh object only on the exact event
  // that caused the break (null on every other), and the projection is memoized
  // on the cursor — so depending on the object itself fires exactly once per
  // break, including when the viewer scrubs backward and crosses it again.
  const [impact, setImpact] = useState<{ attacker: Side; defender: Side } | null>(null)
  const flash = state.breakFlash
  useEffect(() => {
    if (!flash) return
    setImpact({ attacker: flash.actor, defender: flash.actor === 'A' ? 'B' : 'A' })
    const t = setTimeout(() => setImpact(null), 700)
    return () => clearTimeout(t)
  }, [flash])

  const winnerId = replay.atEnd || state.finished ? record.outcome?.winner_id : ''
  const isAD = man.format === 'attack_defense'

  return (
    <div className="relative flex min-h-0 flex-col gap-4">
      {/* BROKE DEFENDER impact wash */}
      <AnimatePresence>
        {impact && (
          <motion.div
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
          shaking={impact?.defender === 'A'}
        />

        <div className="hidden items-center justify-center lg:flex">
          <motion.div
            animate={impact ? { scale: [1, 1.35, 1], rotate: [0, -12, 0] } : {}}
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
          shaking={impact?.defender === 'B'}
        />
      </div>

      <ReplayControls replay={replay} events={record.events ?? []} />

      <AnimatePresence>
        {(replay.atEnd || state.finished) && record.outcome && (
          <OutcomeBanner outcome={record.outcome} />
        )}
      </AnimatePresence>

      <div className="min-h-[220px] flex-1">
        <EventFeed feed={state.feed} />
      </div>
    </div>
  )
}
