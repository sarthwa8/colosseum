import { AnimatePresence, motion } from 'motion/react'
import { useEffect, useRef } from 'react'
import type { FeedEntry } from '../../replay/state'
import VerdictBadge from '../ui/VerdictBadge'

const ACTOR_CLASS = {
  A: 'text-fighter-a border-fighter-a/30',
  B: 'text-fighter-b border-fighter-b/30',
} as const

const KIND_MARK: Record<FeedEntry['kind'], string> = {
  phase: '▸',
  submission: '→',
  attack: '⚑',
  break: '⚔',
  shield: '⛨',
  forfeit: '✗',
  note: '·',
}

const KIND_CLASS: Record<FeedEntry['kind'], string> = {
  phase: 'text-gold',
  submission: 'text-ink-dim',
  attack: 'text-gold',
  break: 'text-pass',
  shield: 'text-ink-dim',
  forfeit: 'text-fail',
  note: 'text-ink-faint',
}

/** Broadcast-style killfeed: newest at the bottom, auto-scrolled. */
export default function EventFeed({ feed }: { feed: FeedEntry[] }) {
  const scroller = useRef<HTMLDivElement>(null)

  // Only follow the tail if the reader is already at it. Force-scrolling on
  // every event yanks anyone who scrolled up to read an earlier entry while
  // playback continues.
  useEffect(() => {
    const el = scroller.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    if (distanceFromBottom < 48) el.scrollTop = el.scrollHeight
  }, [feed.length])

  return (
    <div className="flex h-full flex-col overflow-hidden rounded-xl border border-arena-line bg-arena-surface/60">
      <div className="flex items-center gap-2 border-b border-arena-line px-3 py-2">
        <h2 className="text-[10px] font-bold uppercase tracking-[0.18em] text-ink-faint">
          Event log
        </h2>
        <span className="ml-auto font-mono text-[10px] text-ink-faint">{feed.length}</span>
      </div>

      <div ref={scroller} className="flex-1 overflow-y-auto px-3 py-2">
        {feed.length === 0 ? (
          <p className="py-6 text-center text-xs text-ink-dim">waiting for first event…</p>
        ) : (
          /* role="log" tells assistive tech this is an append-only stream, so
             new entries are read incrementally rather than re-reading the lot. */
          <ul role="log" className="space-y-1">
            <AnimatePresence initial={false}>
              {feed.map(entry => (
                <motion.li
                  key={entry.seq}
                  initial={{ opacity: 0, x: -8 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.18 }}
                  className="flex items-start gap-2 text-xs leading-relaxed"
                >
                  <span className={`mt-px shrink-0 ${KIND_CLASS[entry.kind]}`}>
                    {KIND_MARK[entry.kind]}
                  </span>
                  {entry.actor && (
                    <span
                      className={`shrink-0 rounded border px-1 font-mono text-[10px] font-bold ${ACTOR_CLASS[entry.actor]}`}
                    >
                      {entry.actor}
                    </span>
                  )}
                  <span
                    className={`min-w-0 break-words ${
                      entry.kind === 'break' ? 'font-bold text-pass' : 'text-ink-dim'
                    }`}
                  >
                    {entry.text}
                  </span>
                  {entry.verdict && (
                    <span className="ml-auto shrink-0">
                      <VerdictBadge verdict={entry.verdict} />
                    </span>
                  )}
                </motion.li>
              ))}
            </AnimatePresence>
          </ul>
        )}
      </div>
    </div>
  )
}
