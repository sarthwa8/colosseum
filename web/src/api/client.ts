import { useEffect, useState } from 'react'
import type { MatchRecord, MatchSummary, Report } from './types'

async function getJSON<T>(url: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(url, { signal })
  if (!res.ok) throw new Error(`${url} → ${res.status}`)
  return res.json() as Promise<T>
}

export const api = {
  matches: (signal?: AbortSignal) => getJSON<MatchSummary[]>('/api/matches', signal),
  match: (id: string, signal?: AbortSignal) => getJSON<MatchRecord>(`/api/matches/${id}`, signal),
  report: (signal?: AbortSignal) => getJSON<Report>('/api/report', signal),
}

/**
 * Poll an endpoint on an interval. The server writes match records as files, so
 * a running ladder shows up without a websocket — polling is the honest fit for
 * a filesystem-backed API.
 */
export function usePolled<T>(
  fetcher: (signal?: AbortSignal) => Promise<T>,
  intervalMs: number,
  deps: unknown[] = [],
) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<Error | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const controller = new AbortController()

    const tick = async () => {
      try {
        const next = await fetcher(controller.signal)
        if (!cancelled) {
          setData(next)
          setError(null)
        }
      } catch (e) {
        if (!cancelled && (e as Error).name !== 'AbortError') setError(e as Error)
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    tick()
    const id = setInterval(tick, intervalMs)
    return () => {
      cancelled = true
      controller.abort()
      clearInterval(id)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return { data, error, loading }
}
