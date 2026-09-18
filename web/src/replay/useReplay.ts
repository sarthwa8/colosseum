import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { MatchRecord } from '../api/types'
import { pacing, project, type ReplayState } from './state'

/**
 * Playback rate. There is deliberately no 0 here: modelling "skip to end" as a
 * speed made Restart a no-op (rewind to 0, then the playback effect instantly
 * re-drained to the end) and did nothing at all while paused. Skipping is an
 * action — `skipToEnd` — not a rate.
 */
export type Speed = 1 | 2 | 4

export interface Replay {
  state: ReplayState
  cursor: number
  total: number
  playing: boolean
  speed: Speed
  atEnd: boolean
  play: () => void
  pause: () => void
  toggle: () => void
  restart: () => void
  skipToEnd: () => void
  seek: (cursor: number) => void
  setSpeed: (s: Speed) => void
}

/**
 * Drives an event-log replay. State is a pure fold over events[0..cursor], so
 * seeking anywhere is just setting the cursor — no re-simulation bookkeeping,
 * no DOM mutation to unwind.
 */
export function useReplay(rec: MatchRecord | null): Replay {
  const events = useMemo(() => rec?.events ?? [], [rec])
  const total = events.length

  const [cursor, setCursor] = useState(0)
  const [playing, setPlaying] = useState(false)
  const [speed, setSpeed] = useState<Speed>(4)
  const timer = useRef<number | null>(null)

  // New record → rewind and autoplay.
  useEffect(() => {
    setCursor(0)
    setPlaying(total > 0)
  }, [rec, total])

  const clear = () => {
    if (timer.current !== null) {
      window.clearTimeout(timer.current)
      timer.current = null
    }
  }

  useEffect(() => {
    clear()
    if (!playing || cursor >= total) return

    // The cursor counts events already applied, so the frame on screen ends at
    // events[cursor-1] and the one we're about to reveal is events[cursor].
    // Pacing must measure that gap — reading [cursor, cursor+1] held for the
    // wrong interval and gave beat events their emphasis one frame late.
    const shown = cursor > 0 ? events[cursor - 1] : events[0]
    const delay = cursor > 0 ? pacing(shown, events[cursor], speed) : 0
    timer.current = window.setTimeout(() => setCursor(c => Math.min(c + 1, total)), delay)
    return clear
  }, [playing, cursor, total, speed, events])

  // Stop at the end.
  useEffect(() => {
    if (cursor >= total && playing) setPlaying(false)
  }, [cursor, total, playing])

  const state = useMemo(() => (rec ? project(rec, cursor) : null), [rec, cursor])

  const restart = useCallback(() => {
    clear()
    setCursor(0)
    setPlaying(true)
  }, [])

  const skipToEnd = useCallback(() => {
    clear()
    setPlaying(false)
    setCursor(total)
  }, [total])

  const seek = useCallback(
    (c: number) => {
      clear()
      setPlaying(false)
      setCursor(Math.max(0, Math.min(c, total)))
    },
    [total],
  )

  // Transport shortcuts. A replay tool people actually scrub through needs
  // these; guarded so they don't hijack typing in a future search field.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement | null
      if (el && (el.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName))) return
      if (e.metaKey || e.ctrlKey || e.altKey) return

      switch (e.key) {
        case ' ':
          e.preventDefault()
          if (cursor >= total && total > 0) restart()
          else setPlaying(p => !p)
          break
        case 'ArrowRight':
          e.preventDefault()
          seek(cursor + (e.shiftKey ? 10 : 1))
          break
        case 'ArrowLeft':
          e.preventDefault()
          seek(cursor - (e.shiftKey ? 10 : 1))
          break
        case 'Home':
          e.preventDefault()
          restart()
          break
        case 'End':
          e.preventDefault()
          skipToEnd()
          break
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [cursor, total, seek, restart, skipToEnd])

  return {
    state: state ?? project({ manifest: { fighters: {} } } as MatchRecord, 0),
    cursor,
    total,
    playing,
    speed,
    atEnd: cursor >= total && total > 0,
    play: () => setPlaying(true),
    pause: () => {
      clear()
      setPlaying(false)
    },
    toggle: () => setPlaying(p => !p),
    restart,
    skipToEnd,
    seek,
    setSpeed,
  }
}
