import { Pause, Play, RotateCcw } from 'lucide-react'
import { useMemo } from 'react'
import type { MatchEvent } from '../../api/types'
import { side } from '../../api/types'
import type { Replay, Speed } from '../../replay/useReplay'

const SPEEDS: { value: Speed; label: string }[] = [
  { value: 0, label: 'skip' },
  { value: 4, label: '4×' },
  { value: 2, label: '2×' },
  { value: 1, label: '1×' },
]

/** Events worth a tick on the timeline, with relative weight. */
const BEAT: Partial<Record<MatchEvent['type'], { h: number; tone: string }>> = {
  submission: { h: 0.55, tone: 'actor' },
  attack_submitted: { h: 0.7, tone: 'var(--color-gold)' },
  attack_result: { h: 1, tone: 'var(--color-pass)' },
  fighter_forfeit: { h: 0.85, tone: 'var(--color-fail)' },
  phase_started: { h: 0.35, tone: 'var(--color-ink-faint)' },
}

/**
 * Transport + timeline. The timeline marks every dramatic beat so the shape of
 * a match is legible before you press play — a tall green spike is a break.
 * Scrubbing works because replay state is a fold over the event prefix.
 */
export default function ReplayControls({ replay, events }: { replay: Replay; events: MatchEvent[] }) {
  const ticks = useMemo(
    () =>
      events
        .map((e, i) => ({ e, i }))
        .filter(({ e }) => BEAT[e.type])
        .map(({ e, i }) => {
          const spec = BEAT[e.type]!
          const who = side(e.actor)
          const tone =
            spec.tone === 'actor'
              ? who === 'A'
                ? 'var(--color-fighter-a)'
                : who === 'B'
                  ? 'var(--color-fighter-b)'
                  : 'var(--color-ink-faint)'
              : spec.tone
          return { i, h: spec.h, tone }
        }),
    [events],
  )

  const pct = replay.total > 0 ? (replay.cursor / replay.total) * 100 : 0

  return (
    <div className="rounded-xl border border-arena-line bg-arena-surface/60 p-3">
      <div className="flex items-center gap-2">
        <button
          onClick={replay.atEnd ? replay.restart : replay.toggle}
          aria-label={replay.atEnd ? 'Replay' : replay.playing ? 'Pause' : 'Play'}
          className="flex h-8 w-8 items-center justify-center rounded-lg border border-gold/40 bg-gold/10 text-gold transition-colors hover:bg-gold/20"
        >
          {replay.atEnd ? <RotateCcw size={14} /> : replay.playing ? <Pause size={14} /> : <Play size={14} />}
        </button>

        <button
          onClick={replay.restart}
          aria-label="Restart replay"
          className="flex h-8 w-8 items-center justify-center rounded-lg border border-arena-line bg-arena-raised/60 text-ink-dim transition-colors hover:border-gold/40 hover:text-ink"
        >
          <RotateCcw size={13} />
        </button>

        <div className="ml-1 flex items-center gap-1 rounded-lg border border-arena-line bg-arena-raised/40 p-0.5">
          {SPEEDS.map(s => (
            <button
              key={s.label}
              onClick={() => replay.setSpeed(s.value)}
              className={`rounded px-2 py-1 text-[11px] font-medium transition-colors ${
                replay.speed === s.value
                  ? 'bg-gold/15 text-gold'
                  : 'text-ink-faint hover:text-ink-dim'
              }`}
            >
              {s.label}
            </button>
          ))}
        </div>

        <span className="ml-auto font-mono text-[11px] tabular-nums text-ink-faint">
          {replay.cursor}/{replay.total}
        </span>
      </div>

      {/* Timeline */}
      <div className="relative mt-3 h-9">
        <div className="absolute inset-x-0 top-4 h-px bg-arena-line" />

        <div className="absolute inset-x-0 top-0 flex h-8 items-end">
          {ticks.map(({ i, h, tone }) => {
            const left = replay.total > 0 ? (i / replay.total) * 100 : 0
            const reached = i < replay.cursor
            return (
              <span
                key={i}
                className="absolute bottom-1 w-px rounded-full transition-opacity duration-300"
                style={{
                  left: `${left}%`,
                  height: `${8 + h * 16}px`,
                  background: tone,
                  opacity: reached ? 0.95 : 0.22,
                  boxShadow: reached ? `0 0 6px -1px ${tone}` : 'none',
                }}
              />
            )
          })}
        </div>

        {/* Played portion */}
        <div
          className="absolute top-4 h-px bg-gradient-to-r from-gold/40 to-gold transition-[width] duration-150"
          style={{ width: `${pct}%` }}
        />

        {/* Playhead */}
        <div
          className="pointer-events-none absolute top-4 z-10 -translate-x-1/2 -translate-y-1/2"
          style={{ left: `${pct}%` }}
        >
          <div className="h-2.5 w-2.5 rounded-full border-2 border-gold bg-arena-void shadow-[0_0_10px_-1px_var(--color-gold)]" />
        </div>

        <input
          type="range"
          min={0}
          max={replay.total}
          value={replay.cursor}
          onChange={e => replay.seek(Number(e.target.value))}
          aria-label="Scrub replay timeline"
          className="absolute inset-x-0 top-0 h-8 w-full cursor-pointer opacity-0"
        />
      </div>
    </div>
  )
}
