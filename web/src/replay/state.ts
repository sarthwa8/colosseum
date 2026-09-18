import {
  payload,
  side,
  type MatchEvent,
  type MatchRecord,
  type Side,
  type Verdict,
} from '../api/types'

/** One fighter's visible state at a point in the replay. */
export interface FighterState {
  side: Side
  model: string
  status: string
  phase: 'idle' | 'thinking' | 'judged' | 'attacking' | 'forfeit' | 'done'
  iteration: number
  verdicts: Verdict[]
  passed: number
  total: number
  code: string
  broke: boolean
  shielded: boolean
  forfeit: string
}

export interface FeedEntry {
  seq: number
  kind: 'phase' | 'submission' | 'attack' | 'break' | 'shield' | 'forfeit' | 'note'
  actor: Side | null
  text: string
  verdict?: Verdict
}

export interface ReplayState {
  fighters: Record<Side, FighterState>
  feed: FeedEntry[]
  phase: string
  /** Set on the frame a break lands, so the UI can fire its impact effect. */
  breakFlash: { seq: number; actor: Side } | null
  finished: boolean
}

function blankFighter(s: Side, model: string): FighterState {
  return {
    side: s,
    model,
    status: 'standing by',
    phase: 'idle',
    iteration: 0,
    verdicts: [],
    passed: 0,
    total: 0,
    code: '',
    broke: false,
    shielded: false,
    forfeit: '',
  }
}

export function initialState(rec: MatchRecord): ReplayState {
  const f = rec.manifest.fighters || {}
  return {
    fighters: {
      A: blankFighter('A', f.A?.model ?? 'fighter A'),
      B: blankFighter('B', f.B?.model ?? 'fighter B'),
    },
    feed: [],
    phase: '',
    breakFlash: null,
    finished: false,
  }
}

/** Fold one event into the projected state. Pure — this is what makes seeking work. */
export function reduce(state: ReplayState, e: MatchEvent): ReplayState {
  const who = side(e.actor)
  const next: ReplayState = {
    ...state,
    fighters: { A: { ...state.fighters.A }, B: { ...state.fighters.B } },
    feed: state.feed,
    breakFlash: null, // only set on the exact event that caused it
  }
  const push = (entry: Omit<FeedEntry, 'seq'>) => {
    next.feed = [...next.feed, { seq: e.seq, ...entry }]
  }

  switch (e.type) {
    case 'phase_started': {
      const { phase, attacker, defender } = payload.phase(e)
      next.phase = phase
      push({
        kind: 'phase',
        actor: null,
        text:
          phase === 'attack' && attacker
            ? `attack phase — ${attacker} probes ${defender}`
            : `${phase} phase`,
      })
      if (phase === 'attack' && attacker) {
        const a = side(attacker)
        if (a) next.fighters[a].phase = 'attacking'
      }
      break
    }

    case 'fighter_thinking': {
      if (!who) break
      const { iteration } = payload.thinking(e)
      const f = next.fighters[who]
      f.phase = 'thinking'
      f.iteration = iteration
      f.status = `thinking · iteration ${iteration}`
      break
    }

    case 'fighter_code': {
      if (!who) break
      next.fighters[who].code = payload.code(e).code
      break
    }

    case 'submission': {
      if (!who) break
      const { verdict, passed, total } = payload.submission(e)
      const f = next.fighters[who]
      f.phase = 'judged'
      f.verdicts = [...f.verdicts, verdict]
      f.passed = passed
      f.total = total
      f.status = `${verdict} · ${passed}/${total} tests`
      push({ kind: 'submission', actor: who, text: `${passed}/${total} tests`, verdict })
      break
    }

    case 'fighter_progress': {
      if (!who) break
      const { passed, total } = payload.submission(e)
      if (total > 0) {
        next.fighters[who].passed = passed
        next.fighters[who].total = total
      }
      break
    }

    case 'attack_submitted': {
      if (!who) break
      const input = payload.attack(e).input
      next.fighters[who].phase = 'attacking'
      next.fighters[who].status = 'crafting an exploit'
      // U+21B5, not U+23CE — the latter is missing from JetBrains Mono and
      // falls back to a glyph that reads as part of the input.
      push({
        kind: 'attack',
        actor: who,
        text: `submits input ${truncate(input.trim().replace(/\n/g, ' ↵ '), 42)}`,
      })
      break
    }

    case 'attack_result': {
      if (!who) break
      const { broke, detail } = payload.attackResult(e)
      const defender: Side = who === 'A' ? 'B' : 'A'
      if (broke) {
        next.fighters[who].broke = true
        next.breakFlash = { seq: e.seq, actor: who }
        push({ kind: 'break', actor: who, text: detail || 'broke the defender' })
      } else {
        next.fighters[defender].shielded = true
        push({ kind: 'shield', actor: who, text: detail || 'defender held' })
      }
      break
    }

    case 'fighter_forfeit': {
      if (!who) break
      const { reason } = payload.forfeit(e)
      const f = next.fighters[who]
      f.phase = 'forfeit'
      f.forfeit = reason
      f.status = `forfeit — ${reason}`
      push({ kind: 'forfeit', actor: who, text: `forfeits: ${reason}` })
      break
    }

    case 'commentary': {
      const text = payload.text(e)
      if (text) push({ kind: 'note', actor: who, text })
      break
    }

    case 'match_finished': {
      next.finished = true
      // Settle each fighter into a terminal status. Without this the card keeps
      // whatever transient line it had mid-action ("crafting an exploit") long
      // after the match is over.
      for (const s of ['A', 'B'] as const) {
        const f = next.fighters[s]
        f.phase = 'done'
        if (f.forfeit) continue
        const last = f.verdicts[f.verdicts.length - 1]
        if (!last) f.status = 'no submission'
        else if (last === 'AC') f.status = `solved · ${f.passed}/${f.total} tests`
        else f.status = `${last} · ${f.passed}/${f.total} tests`
      }
      break
    }

    default:
      break
  }

  return next
}

/** Project the match state at a given cursor (number of events applied). */
export function project(rec: MatchRecord, cursor: number): ReplayState {
  const evs = rec.events ?? []
  let s = initialState(rec)
  for (let i = 0; i < Math.min(cursor, evs.length); i++) s = reduce(s, evs[i])
  return s
}

function truncate(s: string, n: number) {
  return s.length <= n ? s : s.slice(0, n) + '…'
}

/**
 * Pacing: how long to hold on an event before advancing.
 *
 * The problem: real match timings are wildly uneven — a `fighter_thinking`
 * event can sit 45 seconds ahead of its `submission` while an LLM works, then
 * three events land in the same millisecond. Playing back proportionally is
 * unwatchable; playing back at a fixed beat throws away the drama of a long
 * think.
 *
 * So we compress logarithmically: short gaps stay roughly proportional, long
 * waits get squeezed toward the ceiling. Beat events (a break landing, a
 * verdict) get a deliberate hold so the eye can catch them.
 */
export function pacing(prev: MatchEvent, next: MatchEvent | undefined, speed: number): number {
  if (!next || speed === 0) return 0
  const gapMs = Math.max(0, new Date(next.at).getTime() - new Date(prev.at).getTime())
  // log-compress: 0ms→90ms, 1s→~420ms, 45s→~900ms
  const compressed = 90 + 260 * Math.log10(1 + gapMs / 120)
  const held = BEAT_EVENTS.has(prev.type) ? Math.max(compressed, 520) : compressed
  return Math.min(1400, held / speed)
}

const BEAT_EVENTS = new Set(['submission', 'attack_result', 'attack_submitted', 'phase_started'])
