// Mirrors the Go JSON contract exactly (internal/match, internal/events,
// internal/rank, internal/web/server.go). The event log is the single source of
// truth for a match; every view here is a projection of it.

export type Verdict = 'AC' | 'WA' | 'RE' | 'CE' | 'TLE' | 'MLE' | 'OLE'

export type EventType =
  | 'match_scheduled'
  | 'match_started'
  | 'phase_started'
  | 'fighter_thinking'
  | 'fighter_code'
  | 'submission'
  | 'fighter_progress'
  | 'fighter_forfeit'
  | 'attack_submitted'
  | 'attack_result'
  | 'commentary'
  | 'match_finished'

/** Fighter slot id. "" / "system" for non-fighter events. */
export type Side = 'A' | 'B'

export interface MatchEvent {
  seq: number
  match_id: string
  type: EventType
  actor?: string
  at: string
  payload?: Record<string, unknown>
}

export interface FighterInfo {
  id: string
  model: string
  provider?: string
  persona?: string
}

export interface Manifest {
  match_id: string
  format: string
  problem: string
  problem_version?: string
  fighters: Record<string, FighterInfo>
  max_iterations?: number
  token_budget?: number
  created_at: string
  seed?: number
}

export interface FighterScore {
  id: string
  model: string
  solved: boolean
  cases_passed: number
  cases_total: number
  iterations: number
  tokens_in: number
  tokens_out: number
  cost_usd: number
  wall_ms: number
  forfeit?: string
  code?: string
  survived?: boolean
  broke?: boolean
}

export interface Outcome {
  winner_id: string // "" => draw
  reason: string
  scores: Record<string, FighterScore>
}

export interface MatchRecord {
  manifest: Manifest
  outcome: Outcome
  events: MatchEvent[]
}

/** Row returned by GET /api/matches. */
export interface MatchSummary {
  id: string
  format: string
  problem: string
  winner: string // model id, or "" for draw
  reason: string
  models: string[]
  created: string
}

export interface EloCI {
  rating: number
  low: number
  high: number
}

export interface ModelRow {
  model: string
  games: number
  wins: number
  losses: number
  draws: number
  solved: number
  solve_games: number
  solve_rate: number
  avg_tokens: number
  ad_games: number
  survivals: number
  breaks: number
  survive_rate: number
  break_rate: number
  cost_usd: number
}

export interface Report {
  total_matches: number
  total_cost_usd: number
  models: ModelRow[]
  race_elo: Record<string, EloCI>
  robustness_elo: Record<string, EloCI>
  win_matrix: Record<string, Record<string, number>>
}

// ---------------------------------------------------------------------------
// Payload accessors. Payload is an open map on the Go side so new event kinds
// don't force schema churn; we narrow at this edge and trust the values above.
// ---------------------------------------------------------------------------

const str = (p: Record<string, unknown> | undefined, k: string): string =>
  typeof p?.[k] === 'string' ? (p[k] as string) : ''

const num = (p: Record<string, unknown> | undefined, k: string): number =>
  typeof p?.[k] === 'number' ? (p[k] as number) : 0

const bool = (p: Record<string, unknown> | undefined, k: string): boolean =>
  p?.[k] === true

export const payload = {
  phase: (e: MatchEvent) => ({
    phase: str(e.payload, 'phase'),
    attacker: str(e.payload, 'attacker'),
    defender: str(e.payload, 'defender'),
  }),
  thinking: (e: MatchEvent) => ({ iteration: num(e.payload, 'iteration') }),
  code: (e: MatchEvent) => ({ code: str(e.payload, 'code') }),
  submission: (e: MatchEvent) => ({
    verdict: (str(e.payload, 'verdict') || 'WA') as Verdict,
    passed: num(e.payload, 'passed'),
    total: num(e.payload, 'total'),
  }),
  attack: (e: MatchEvent) => ({ input: str(e.payload, 'input') }),
  attackResult: (e: MatchEvent) => ({
    broke: bool(e.payload, 'broke'),
    detail: str(e.payload, 'detail'),
  }),
  forfeit: (e: MatchEvent) => ({ reason: str(e.payload, 'reason') }),
  title: (e: MatchEvent) => str(e.payload, 'title'),
  text: (e: MatchEvent) => str(e.payload, 'text'),
}

/** Narrow an actor string to a fighter side, or null for system events. */
export function side(actor: string | undefined): Side | null {
  return actor === 'A' || actor === 'B' ? actor : null
}

export const VERDICT_TONE: Record<Verdict, 'pass' | 'fail' | 'limit'> = {
  AC: 'pass',
  WA: 'fail',
  RE: 'fail',
  CE: 'fail',
  TLE: 'limit',
  MLE: 'limit',
  OLE: 'limit',
}

export const VERDICT_LABEL: Record<Verdict, string> = {
  AC: 'Accepted',
  WA: 'Wrong Answer',
  RE: 'Runtime Error',
  CE: 'Compile Error',
  TLE: 'Time Limit',
  MLE: 'Memory Limit',
  OLE: 'Output Limit',
}
