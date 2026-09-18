import { useEffect, useState } from 'react'
import { motion } from 'motion/react'
import { api, usePolled } from './api/client'
import type { MatchRecord } from './api/types'
import Arena from './components/arena/Arena'
import Particles from './components/background/Particles'
import Leaderboard from './components/sidebar/Leaderboard'
import MatchList from './components/sidebar/MatchList'
import ShinyText from './components/text/ShinyText'

export default function App() {
  const { data: matches } = usePolled(api.matches, 5000)
  const { data: report } = usePolled(api.report, 5000)

  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [record, setRecord] = useState<MatchRecord | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)

  // Default to the newest match once the list arrives.
  useEffect(() => {
    if (!selectedId && matches && matches.length > 0) setSelectedId(matches[0].id)
  }, [matches, selectedId])

  useEffect(() => {
    if (!selectedId) return
    let cancelled = false
    setLoadError(null)
    api
      .match(selectedId)
      .then(r => !cancelled && setRecord(r))
      .catch(e => !cancelled && setLoadError((e as Error).message))
    return () => {
      cancelled = true
    }
  }, [selectedId])

  return (
    <div className="arena-floor grid-veil relative min-h-screen">
      {/* Ambient WebGL dust — fixed so it never scrolls with content. */}
      <Particles className="pointer-events-none fixed inset-0 z-0 opacity-70" />

      <div className="relative z-10 flex min-h-screen flex-col">
        <header className="sticky top-0 z-20 border-b border-arena-line/80 bg-arena-void/80 backdrop-blur-md">
          <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1 px-5 py-3">
            <h1 className="font-display text-lg font-bold tracking-[0.22em]">
              <span className="text-gold">⚔</span>{' '}
              <ShinyText text="COLOSSEUM" speed={6} />
            </h1>
            <p className="text-[11px] text-ink-faint">
              two models write code · and try to break each other&apos;s
            </p>

            <div className="ml-auto flex items-center gap-4 font-mono text-[11px] text-ink-faint">
              {report && report.total_matches > 0 && (
                <>
                  <span>
                    <span className="text-ink-dim">{report.total_matches}</span> matches
                  </span>
                  {report.total_cost_usd > 0 && (
                    <span>
                      <span className="text-ink-dim">${report.total_cost_usd.toFixed(2)}</span> spent
                    </span>
                  )}
                </>
              )}
              <a
                href="https://github.com/sarthwa8/colosseum"
                target="_blank"
                rel="noreferrer"
                className="transition-colors hover:text-gold"
              >
                source ↗
              </a>
            </div>
          </div>
        </header>

        <div className="flex flex-1 flex-col gap-4 p-4 lg:grid lg:grid-cols-[300px_1fr] lg:items-start">
          <aside className="flex max-h-[calc(100vh-5.5rem)] flex-col gap-4 overflow-y-auto lg:sticky lg:top-[4.25rem]">
            <section>
              <SectionTitle>Standings</SectionTitle>
              <Leaderboard report={report} />
            </section>
            <section>
              <SectionTitle>Matches</SectionTitle>
              <MatchList
                matches={matches ?? []}
                selectedId={selectedId}
                onSelect={setSelectedId}
              />
            </section>
          </aside>

          <main className="min-w-0">
            {loadError ? (
              <EmptyState title="Could not load that match" detail={loadError} />
            ) : record ? (
              <Arena key={record.manifest.match_id} record={record} />
            ) : matches && matches.length === 0 ? (
              <EmptyState
                title="The arena is empty"
                detail="Run a match to fill it: colosseum match --problem eval-expr --a mock:reference --b mock:wrong --format ad"
              />
            ) : (
              <EmptyState title="Loading the arena…" detail="" />
            )}
          </main>
        </div>
      </div>
    </div>
  )
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="mb-2 px-1 text-[10px] font-bold uppercase tracking-[0.18em] text-ink-faint">
      {children}
    </h2>
  )
}

function EmptyState({ title, detail }: { title: string; detail: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="mt-24 text-center"
    >
      <div className="mb-3 text-3xl opacity-30">⚔</div>
      <h2 className="font-display text-lg font-bold text-ink-dim">{title}</h2>
      {detail && (
        <p className="mx-auto mt-2 max-w-md break-words px-4 font-mono text-[11px] leading-relaxed text-ink-faint">
          {detail}
        </p>
      )}
    </motion.div>
  )
}
