// Package problem loads versioned competitive-programming problems from disk.
// A problem bundles its statement, hidden test cases, resource limits, input
// constraints (used by the attack/defense format), and a pinned reference
// solution (the oracle that validates attacker-proposed inputs).
package problem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sarthaksukhral/colosseum/internal/judge"
)

// Problem is a fully-loaded problem ready to judge.
type Problem struct {
	Slug        string
	Title       string
	Difficulty  string // easy | medium | hard
	Version     string // bumped when cases/statement change; pins reproducibility
	Statement   string // markdown
	Constraints string // human/LLM-readable input constraints for attackers
	Limits      judge.Limits
	Cases       []judge.Case
	Reference   string // pinned reference solution source
	RefLang     string
}

// manifest is the on-disk problem.json schema.
type manifest struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Difficulty  string `json:"difficulty"`
	Version     string `json:"version"`
	Statement   string `json:"statement"`   // relative path, default statement.md
	Constraints string `json:"constraints"` // relative path, optional
	Reference   string `json:"reference"`   // relative path to reference solution
	RefLang     string `json:"reference_lang"`
	Limits      struct {
		WallSeconds float64 `json:"wall_seconds"`
		MemoryMiB   int64   `json:"memory_mib"`
	} `json:"limits"`
}

// Load reads a single problem directory.
func Load(dir string) (*Problem, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "problem.json"))
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	p := &Problem{
		Slug:       m.Slug,
		Title:      m.Title,
		Difficulty: m.Difficulty,
		Version:    m.Version,
		RefLang:    m.RefLang,
	}
	if p.RefLang == "" {
		p.RefLang = "python"
	}
	if p.Version == "" {
		p.Version = "1"
	}

	p.Statement = readOptional(dir, orDefault(m.Statement, "statement.md"))
	p.Constraints = readOptional(dir, m.Constraints)
	if m.Reference != "" {
		p.Reference = readOptional(dir, m.Reference)
	}

	p.Limits = judge.Limits{}
	if m.Limits.WallSeconds > 0 {
		p.Limits.Wall = time.Duration(m.Limits.WallSeconds * float64(time.Second))
	}
	p.Limits.MemoryMiB = m.Limits.MemoryMiB

	cases, err := loadCases(filepath.Join(dir, "cases"))
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("problem %q has no test cases", m.Slug)
	}
	p.Cases = cases
	return p, nil
}

// LoadAll loads every problem subdirectory under root, sorted by slug.
func LoadAll(root string) ([]*Problem, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var probs []*Problem
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := Load(filepath.Join(root, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", e.Name(), err)
		}
		probs = append(probs, p)
	}
	sort.Slice(probs, func(i, j int) bool { return probs[i].Slug < probs[j].Slug })
	return probs, nil
}

// loadCases pairs NN.in with NN.out files in a cases directory.
func loadCases(dir string) ([]judge.Case, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading cases dir: %w", err)
	}
	var cases []judge.Case
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".in") {
			continue
		}
		base := strings.TrimSuffix(name, ".in")
		in, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		out, err := os.ReadFile(filepath.Join(dir, base+".out"))
		if err != nil {
			return nil, fmt.Errorf("case %s missing .out: %w", base, err)
		}
		cases = append(cases, judge.Case{Name: base, Input: string(in), Expected: string(out)})
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
	return cases, nil
}

func readOptional(dir, rel string) string {
	if rel == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return ""
	}
	return string(b)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
