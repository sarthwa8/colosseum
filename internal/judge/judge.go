package judge

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Judge orchestrates checking and running a submission across its test cases.
type Judge struct {
	Runner  Runner
	Workers int // max concurrent case containers
}

// New returns a Judge backed by runner with a sane default parallelism.
func New(runner Runner) *Judge {
	return &Judge{Runner: runner, Workers: 4}
}

// Judge runs sub to completion and returns the aggregate report.
func (j *Judge) Judge(ctx context.Context, sub Submission) (Report, error) {
	return j.JudgeStream(ctx, sub, nil)
}

// JudgeStream is like Judge but also emits each CaseResult on progress as it
// completes (nil channel => no streaming). progress is never closed by Judge.
func (j *Judge) JudgeStream(ctx context.Context, sub Submission, progress chan<- CaseResult) (Report, error) {
	lang, err := LookupLanguage(sub.Language)
	if err != nil {
		return Report{Verdict: InternalErr}, err
	}

	dir, err := os.MkdirTemp("", "colosseum-work-")
	if err != nil {
		return Report{Verdict: InternalErr}, err
	}
	defer os.RemoveAll(dir)

	// World-accessible so the container's nobody user can read (and, during the
	// writable check phase, write). The dir is a throwaway temp.
	if err := os.WriteFile(filepath.Join(dir, lang.Source), []byte(sub.Code), 0o644); err != nil {
		return Report{Verdict: InternalErr}, err
	}
	if err := chmodTree(dir, 0o777); err != nil {
		return Report{Verdict: InternalErr}, err
	}

	// Check / compile phase (workdir writable).
	if len(lang.Check) > 0 {
		out, err := j.Runner.Run(ctx, RunSpec{
			Image:    lang.Image,
			Argv:     lang.Check,
			HostDir:  dir,
			Writable: true,
			Limits:   sub.Limits,
		})
		if err != nil {
			return Report{Verdict: InternalErr}, err
		}
		if out.ExitCode != 0 || out.TimedOut {
			cr := &CaseResult{
				Verdict: CompileError,
				Stderr:  out.Stderr,
				Detail:  "compilation/syntax check failed",
			}
			return Report{Verdict: CompileError, Total: len(sub.Cases), Compile: cr}, nil
		}
	}

	results := j.runCases(ctx, lang, dir, sub, progress)
	return aggregate(results), nil
}

// runCases executes every case with bounded parallelism.
func (j *Judge) runCases(ctx context.Context, lang Language, dir string, sub Submission, progress chan<- CaseResult) []CaseResult {
	workers := j.Workers
	if workers < 1 {
		workers = 1
	}
	results := make([]CaseResult, len(sub.Cases))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i, c := range sub.Cases {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, c Case) {
			defer wg.Done()
			defer func() { <-sem }()
			cr := j.runCase(ctx, lang, dir, c, i, sub.Limits)
			results[i] = cr
			if progress != nil {
				select {
				case progress <- cr:
				case <-ctx.Done():
				}
			}
		}(i, c)
	}
	wg.Wait()
	return results
}

// runCase runs one test case and classifies the outcome.
func (j *Judge) runCase(ctx context.Context, lang Language, dir string, c Case, idx int, lim Limits) CaseResult {
	out, err := j.Runner.Run(ctx, RunSpec{
		Image:   lang.Image,
		Argv:    lang.Run,
		Stdin:   c.Input,
		HostDir: dir,
		Limits:  lim,
	})
	cr := CaseResult{Index: idx, Name: c.Name}
	if err != nil {
		cr.Verdict = InternalErr
		cr.Detail = err.Error()
		return cr
	}
	cr.Stdout = out.Stdout
	cr.Stderr = out.Stderr
	cr.ExitCode = out.ExitCode
	cr.Duration = out.Duration
	cr.Truncated = out.Truncated
	cr.Verdict = classify(out, c.Expected)
	return cr
}

// classify maps a raw run to a verdict. Order matters: limit kills are checked
// before exit-code inspection because a killed process's exit code is ambiguous.
func classify(out RunOutput, expected string) Verdict {
	switch {
	case out.TimedOut:
		return TimeLimit
	case out.OOMKilled:
		return MemoryLimit
	case out.Truncated:
		return OutputLimit
	case out.ExitCode != 0:
		return RuntimeError
	case normalizeOutput(out.Stdout) == normalizeOutput(expected):
		return Accepted
	default:
		return WrongAnswer
	}
}

// aggregate collapses per-case results into a report. Overall verdict is AC iff
// every case passed, otherwise the first non-AC verdict in case order.
func aggregate(results []CaseResult) Report {
	sort.Slice(results, func(a, b int) bool { return results[a].Index < results[b].Index })
	rep := Report{Verdict: Accepted, Total: len(results), Cases: results}
	for _, r := range results {
		if r.Verdict == Accepted {
			rep.Passed++
			continue
		}
		if rep.Verdict == Accepted {
			rep.Verdict = r.Verdict
		}
	}
	if len(results) == 0 {
		rep.Verdict = Accepted
	}
	return rep
}

// SameOutput reports whether two program outputs are equal under the same
// leniency the judge uses for verdicts. Used by the Attack/Defense format to
// compare a defender's output against the reference oracle.
func SameOutput(a, b string) bool {
	return normalizeOutput(a) == normalizeOutput(b)
}

// RunInput executes code on a single stdin input and reports its stdout plus
// whether it "ran cleanly" (exited 0 within limits — verdict AC or WA, since
// there's no expected output to compare). Attack/Defense uses this to run both
// the reference oracle and the defender on an attacker-proposed input. A false
// clean means the program crashed, timed out, or blew a limit.
func (j *Judge) RunInput(ctx context.Context, lang, code, input string, lim Limits) (stdout string, clean bool, verdict Verdict, err error) {
	rep, err := j.Judge(ctx, Submission{
		Language: lang,
		Code:     code,
		Cases:    []Case{{Name: "input", Input: input, Expected: ""}},
		Limits:   lim,
	})
	if err != nil {
		return "", false, InternalErr, err
	}
	if rep.Verdict == CompileError {
		return "", false, CompileError, nil
	}
	if len(rep.Cases) == 0 {
		return "", false, InternalErr, nil
	}
	c := rep.Cases[0]
	clean = c.Verdict == Accepted || c.Verdict == WrongAnswer
	return c.Stdout, clean, c.Verdict, nil
}

// normalizeOutput applies standard competitive-judging leniency: trailing
// whitespace on each line is dropped and trailing blank lines are ignored, so
// cosmetic newline differences don't cause false WRONG_ANSWERs.
func normalizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func chmodTree(root string, mode os.FileMode) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, mode)
	})
}
