package events

import (
	"context"
	"time"
)

// Replay walks a stored event slice and calls emit for each, preserving the
// original inter-event timing scaled by speed (2.0 = twice as fast). speed <= 0
// means no delay (instant). This is the replay cursor: live spectating and
// after-the-fact replay render the exact same event stream from the same source
// — Replay just re-establishes the timeline. Honors ctx cancellation.
func Replay(ctx context.Context, evs []Event, speed float64, emit func(Event)) error {
	var prev time.Time
	for i, e := range evs {
		if i > 0 && speed > 0 {
			gap := e.At.Sub(prev)
			if gap > 0 {
				scaled := time.Duration(float64(gap) / speed)
				// Cap any single gap so a long think doesn't stall a replay.
				if scaled > 3*time.Second {
					scaled = 3 * time.Second
				}
				select {
				case <-time.After(scaled):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		emit(e)
		prev = e.At
	}
	return nil
}
