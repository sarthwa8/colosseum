// Package web serves the spectator UI: an embedded single-page app that lists
// saved matches and replays each one from its event log in the browser — the
// same event stream the terminal renders, on the same data the ladder scores.
// No build step; the page is go:embed'd into the binary.
package web

import (
	"embed"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sarthaksukhral/colosseum/internal/match"
)

//go:embed index.html
var content embed.FS

// Server serves match records from a data directory.
type Server struct {
	dataDir string
}

func NewServer(dataDir string) *Server { return &Server{dataDir: dataDir} }

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.index)
	mux.HandleFunc("/api/matches", s.listMatches)
	mux.HandleFunc("/api/matches/", s.getMatch)
	mux.HandleFunc("/api/report", s.report)
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, err := content.ReadFile("index.html")
	if err != nil {
		http.Error(w, "index missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

// matchSummary is the list-view row for one saved match.
type matchSummary struct {
	ID      string   `json:"id"`
	Format  string   `json:"format"`
	Problem string   `json:"problem"`
	Winner  string   `json:"winner"` // model id, or "" for draw
	Reason  string   `json:"reason"`
	Models  []string `json:"models"`
	Created string   `json:"created"`
}

func (s *Server) listMatches(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		writeJSON(w, []matchSummary{}) // empty is fine (no matches yet)
		return
	}
	var out []matchSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		rec, err := match.LoadRecord(filepath.Join(s.dataDir, e.Name()))
		if err != nil {
			continue
		}
		sum := matchSummary{
			ID:      rec.Manifest.MatchID,
			Format:  rec.Manifest.Format,
			Problem: rec.Manifest.Problem,
			Reason:  rec.Outcome.Reason,
			Created: rec.Manifest.CreatedAt.Format("2006-01-02 15:04"),
		}
		for _, f := range rec.Manifest.Fighters {
			sum.Models = append(sum.Models, f.Model)
		}
		sort.Strings(sum.Models)
		if rec.Outcome.WinnerID != "" {
			sum.Winner = rec.Outcome.Scores[rec.Outcome.WinnerID].Model
		}
		out = append(out, sum)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	writeJSON(w, out)
}

func (s *Server) getMatch(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/matches/")
	// Guard against path traversal — ids are hex tokens.
	if id == "" || strings.ContainsAny(id, "/\\.") {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	b, err := os.ReadFile(filepath.Join(s.dataDir, id+".json"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile(filepath.Join(filepath.Dir(s.dataDir), "report.json"))
	if err != nil {
		writeJSON(w, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
