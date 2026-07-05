package events

import "testing"

func TestAppendAssignsSequence(t *testing.T) {
	l := NewLog("m1")
	e1 := l.Emit(MatchStarted, "system", nil)
	e2 := l.Emit(Submission, "A", map[string]any{"verdict": "AC"})
	if e1.Seq != 1 || e2.Seq != 2 {
		t.Fatalf("sequence not monotonic: %d %d", e1.Seq, e2.Seq)
	}
	if e1.MatchID != "m1" {
		t.Errorf("match id not stamped: %q", e1.MatchID)
	}
	if e1.At.IsZero() {
		t.Errorf("timestamp not stamped")
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	l := NewLog("m1")
	l.Emit(MatchStarted, "system", nil)
	snap := l.Snapshot()
	l.Emit(MatchFinished, "system", nil)
	if len(snap) != 1 {
		t.Fatalf("snapshot should be a point-in-time copy, got %d", len(snap))
	}
}

// Subscribe(from) is the reconnect primitive: a client that dropped at seq N
// resubscribes from N+1 and must receive exactly the missed events plus live
// ones — no gaps, no dupes.
func TestSubscribeReplaysFromSeq(t *testing.T) {
	l := NewLog("m1")
	l.Emit(MatchStarted, "system", nil)   // seq 1
	l.Emit(Submission, "A", nil)          // seq 2
	l.Emit(Submission, "B", nil)          // seq 3

	// Reconnect from seq 3 onward.
	ch, cancel := l.Subscribe(3)
	defer cancel()

	// Backlog: only seq 3 (>= from).
	got := <-ch
	if got.Seq != 3 {
		t.Fatalf("expected backlog to start at seq 3, got %d", got.Seq)
	}

	// A live event is delivered too.
	l.Emit(MatchFinished, "system", nil) // seq 4
	got = <-ch
	if got.Seq != 4 {
		t.Fatalf("expected live event seq 4, got %d", got.Seq)
	}
}

func TestSubscribeFromStartReplaysAll(t *testing.T) {
	l := NewLog("m1")
	for i := 0; i < 5; i++ {
		l.Emit(Submission, "A", nil)
	}
	ch, cancel := l.Subscribe(1)
	defer cancel()
	for i := 1; i <= 5; i++ {
		if got := <-ch; got.Seq != i {
			t.Fatalf("expected seq %d, got %d", i, got.Seq)
		}
	}
}

func TestCancelStopsDelivery(t *testing.T) {
	l := NewLog("m1")
	ch, cancel := l.Subscribe(1)
	cancel()
	// Channel is closed; a receive returns the zero value with ok=false.
	if _, ok := <-ch; ok {
		t.Errorf("expected channel closed after cancel")
	}
}
