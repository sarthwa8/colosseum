import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { MatchRecord } from '../api/types'
import { pacing, project, type ReplayState } from './state'

export type Speed = 0 | 1 | 2 | 4

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

    // speed 0 = "instant": drain the whole log in one frame.
    if (speed === 0) {
      setCursor(total)
      setPlaying(false)
      return
    }

    const delay = pacing(events[cursor], events[cursor + 1], speed)
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

  const seek = useCallback(
    (c: number) => {
      clear()
      setPlaying(false)
      setCursor(Math.max(0, Math.min(c, total)))
    },
    [total],
  )

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
    seek,
    setSpeed,
  }
}
