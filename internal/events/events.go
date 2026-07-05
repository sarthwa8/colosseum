// Package events is the event-sourcing backbone. Every match is an append-only
// stream of events; both the live broadcast and after-the-fact replays render
// from the same stream, and the ladder's raw data is exported from it. Nothing
// about a match's history exists outside this log.
package events

import (
	"sync"
	"time"
)

// Type enumerates the kinds of events a match emits.
type Type string

const (
	MatchScheduled  Type = "match_scheduled"
	MatchStarted    Type = "match_started"
	PhaseStarted    Type = "phase_started"
	FighterThinking Type = "fighter_thinking"
	FighterCode     Type = "fighter_code"     // a code attempt was produced
	Submission      Type = "submission"       // a judged result
	FighterProgress Type = "fighter_progress" // cases passed / total after a submission
	FighterForfeit  Type = "fighter_forfeit"  // budget/error/walkover
	AttackSubmitted Type = "attack_submitted" // an attacker proposed a breaking input
	AttackResult    Type = "attack_result"    // whether the attack broke the defender
	Commentary      Type = "commentary"       // AI play-by-play
	MatchFinished   Type = "match_finished"
)

// Event is one immutable entry in a match's log. Payload is an open map so new
// event kinds and frontend fields don't require schema churn; typed helpers in
// this package construct the common ones.
type Event struct {
	Seq     int            `json:"seq"`     // per-match, monotonic from 1
	MatchID string         `json:"match_id"`
	Type    Type           `json:"type"`
	Actor   string         `json:"actor,omitempty"` // fighter id, or "system"
	At      time.Time      `json:"at"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Log is an in-memory append-only event stream with live subscribers. A
// persistent (SQLite-backed) store implements the same Append/Snapshot surface
// in a later milestone; the pub/sub semantics here are the contract.
type Log struct {
	matchID string
	clock   func() time.Time

	mu      sync.Mutex
	events  []Event
	subs    map[int]*subscriber
	nextSub int
}

type subscriber struct {
	ch     chan Event
	cancel bool
}

// NewLog creates an empty log for a match.
func NewLog(matchID string) *Log {
	return &Log{
		matchID: matchID,
		clock:   time.Now,
		subs:    make(map[int]*subscriber),
	}
}

// Append stamps the event with the next sequence number and timestamp, stores
// it, and fans it out to live subscribers. It returns the stored event.
func (l *Log) Append(e Event) Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	e.Seq = len(l.events) + 1
	e.MatchID = l.matchID
	if e.At.IsZero() {
		e.At = l.clock()
	}
	l.events = append(l.events, e)
	for _, s := range l.subs {
		// Buffered channels; a slow subscriber can't stall the match. If full,
		// drop for that subscriber — it can re-sync via Snapshot on reconnect.
		select {
		case s.ch <- e:
		default:
		}
	}
	return e
}

// Emit is a convenience constructor + Append.
func (l *Log) Emit(t Type, actor string, payload map[string]any) Event {
	return l.Append(Event{Type: t, Actor: actor, Payload: payload})
}

// Snapshot returns a copy of all events so far.
func (l *Log) Snapshot() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Event, len(l.events))
	copy(out, l.events)
	return out
}

// Subscribe returns a channel that first replays every event with Seq >= from,
// then delivers live events, plus a cancel func. This is the reconnect-and-
// replay primitive: a client that dropped at seq N resubscribes from N+1 with
// no gaps and no dupes. from <= 1 replays the whole match.
func (l *Log) Subscribe(from int) (<-chan Event, func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	backlog := make([]Event, 0)
	for _, e := range l.events {
		if e.Seq >= from {
			backlog = append(backlog, e)
		}
	}
	ch := make(chan Event, len(backlog)+256)
	for _, e := range backlog {
		ch <- e
	}

	id := l.nextSub
	l.nextSub++
	l.subs[id] = &subscriber{ch: ch}

	cancel := func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if s, ok := l.subs[id]; ok {
			delete(l.subs, id)
			close(s.ch)
		}
	}
	return ch, cancel
}
