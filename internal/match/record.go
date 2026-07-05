package match

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sarthaksukhral/colosseum/internal/events"
)

// Record is the persisted form of a finished match: its manifest, full event
// log, and outcome. This single artifact powers replays (M3), the ladder's raw
// data (M5), and reproducibility audits — the event log is the source of truth.
type Record struct {
	Manifest Manifest       `json:"manifest"`
	Outcome  Outcome        `json:"outcome"`
	Events   []events.Event `json:"events"`
}

// Record snapshots the match into a persistable Record.
func (m *Match) ToRecord(o Outcome) Record {
	return Record{Manifest: m.Manifest, Outcome: o, Events: m.Log.Snapshot()}
}

// SaveRecord writes r to <dir>/<match-id>.json and returns the path.
func SaveRecord(dir string, r Record) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, r.Manifest.MatchID+".json")
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// LoadRecord reads a persisted match record.
func LoadRecord(path string) (Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var r Record
	if err := json.Unmarshal(b, &r); err != nil {
		return Record{}, fmt.Errorf("parsing match record: %w", err)
	}
	return r, nil
}
