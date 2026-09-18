// Package web serves the spectator UI: an embedded single-page app that lists
// saved matches and replays each one from its event log in the browser — the
// same event stream the terminal renders, on the same data the ladder scores.
//
// The UI is a React/Vite bundle built from ../../web into dist/ and go:embed'd
// here, so the binary stays self-contained: no Node toolchain, no separate
// server, `go install` just works. dist/ is committed for that reason — go:embed
// resolves at compile time, so a missing directory is a build error.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sarthaksukhral/colosseum/internal/match"
)

//go:embed all:dist
var content embed.FS

// Server serves match records from a data directory.
type Server struct {
	dataDir string
}

func NewServer(dataDir string) *Server { return &Server{dataDir: dataDir} }

// Handler returns the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.static)
	mux.HandleFunc("/api/matches", s.listMatches)
	mux.HandleFunc("/api/matches/", s.getMatch)
	mux.HandleFunc("/api/report", s.report)
	return mux
}

// static serves the built SPA. Hashed asset filenames are immutable, so they
// get a long cache; index.html must not be cached, since it's the file that
// points at the current hashes.
func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	dist, err := fs.Sub(content, "dist")
	if err != nil {
		http.Error(w, "ui bundle missing", http.StatusInternalServerError)
		return
	}

	// The SPA fallback must not swallow the API surface: an unknown /api/ path
	// has to 404, or a client hitting a bad endpoint gets the HTML shell and a
	// baffling JSON parse error instead of a status it can act on.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "" && name != "." {
		if f, err := dist.Open(name); err == nil {
			f.Close()
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.FileServer(http.FS(dist)).ServeHTTP(w, r)
			return
		}
	}

	// SPA fallback: unknown paths render the app shell, not a 404.
	b, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "ui bundle missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
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
	// Non-nil so an empty arena encodes as [] rather than null — the UI maps
	// over this directly.
	out := []matchSummary{}
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
