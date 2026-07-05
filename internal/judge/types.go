// Package judge executes untrusted code against test cases inside disposable,
// resource-limited Docker containers. It is the security-critical core of
// Colosseum: every line of code it runs was written by an adversarially-prompted
// LLM, so containment is the whole game. See docs/SANDBOX.md for the threat model.
package judge

import "time"

// Verdict is the outcome of running a submission against a single test case
// (or, at the report level, the aggregate outcome).
type Verdict string

const (
	Accepted     Verdict = "AC" // output matched expected
	WrongAnswer  Verdict = "WA" // ran cleanly but output differed
	TimeLimit    Verdict = "TLE"
	RuntimeError Verdict = "RE" // non-zero exit that wasn't a limit kill
	MemoryLimit  Verdict = "MLE"
	OutputLimit  Verdict = "OLE" // exceeded the stdout byte cap
	CompileError Verdict = "CE"
	InternalErr  Verdict = "IE" // the judge itself failed (docker missing, etc.)
)

// Limits are the resource ceilings enforced on a single container run. Zero
// values are replaced by DefaultLimits at execution time.
type Limits struct {
	Wall      time.Duration // hard wall-clock timeout; on expiry the container is SIGKILLed
	MemoryMiB int64         // container memory ceiling (swap disabled) → OOM kill on breach
	CPUs      float64       // fractional CPU quota
	Pids      int64         // max process/thread count → contains fork bombs
	OutputKiB int64         // max stdout captured; overflow → OLE
}

// DefaultLimits are deliberately tight: a toy CP solution needs almost nothing,
// and generous limits only give hostile code more room.
var DefaultLimits = Limits{
	Wall:      10 * time.Second,
	MemoryMiB: 256,
	CPUs:      1.0,
	Pids:      128,
	OutputKiB: 256,
}

// withDefaults fills any zero field from DefaultLimits so callers can specify
// only what they care about.
func (l Limits) withDefaults() Limits {
	if l.Wall == 0 {
		l.Wall = DefaultLimits.Wall
	}
	if l.MemoryMiB == 0 {
		l.MemoryMiB = DefaultLimits.MemoryMiB
	}
	if l.CPUs == 0 {
		l.CPUs = DefaultLimits.CPUs
	}
	if l.Pids == 0 {
		l.Pids = DefaultLimits.Pids
	}
	if l.OutputKiB == 0 {
		l.OutputKiB = DefaultLimits.OutputKiB
	}
	return l
}

// Case is a single test: feed Input on stdin, expect Expected on stdout.
type Case struct {
	Name     string
	Input    string
	Expected string
}

// Submission is one program run against a set of cases.
type Submission struct {
	Language string
	Code     string
	Cases    []Case
	Limits   Limits
}

// CaseResult is the outcome of one test case.
type CaseResult struct {
	Index     int           `json:"index"`
	Name      string        `json:"name"`
	Verdict   Verdict       `json:"verdict"`
	Stdout    string        `json:"stdout,omitempty"`
	Stderr    string        `json:"stderr,omitempty"`
	ExitCode  int           `json:"exit_code"`
	Duration  time.Duration `json:"duration_ns"`
	Truncated bool          `json:"truncated,omitempty"`
	Detail    string        `json:"detail,omitempty"` // human-readable note (why RE/IE, etc.)
}

// Report is the aggregate outcome of judging a submission.
type Report struct {
	Verdict Verdict      `json:"verdict"` // AC iff every case is AC; else the first non-AC verdict
	Passed  int          `json:"passed"`
	Total   int          `json:"total"`
	Cases   []CaseResult `json:"cases"`
	Compile *CaseResult  `json:"compile,omitempty"` // populated on CE
}
