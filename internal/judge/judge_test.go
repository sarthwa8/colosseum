package judge

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeRunner lets us exercise the orchestration/classification logic without
// Docker. It routes on whether the command is the check or the run.
type fakeRunner struct {
	check func(spec RunSpec) (RunOutput, error)
	run   func(spec RunSpec) (RunOutput, error)
}

func (f fakeRunner) Run(_ context.Context, spec RunSpec) (RunOutput, error) {
	// The check phase is the only writable run.
	if spec.Writable {
		if f.check != nil {
			return f.check(spec)
		}
		return RunOutput{ExitCode: 0}, nil
	}
	return f.run(spec)
}

func TestNormalizeOutput(t *testing.T) {
	cases := []struct{ a, b string }{
		{"5\n", "5"},
		{"5  \n", "5"},
		{"1 2 3\n\n\n", "1 2 3"},
		{"a\r\nb\r\n", "a\nb"},
	}
	for _, c := range cases {
		if normalizeOutput(c.a) != normalizeOutput(c.b) {
			t.Errorf("normalize(%q) != normalize(%q)", c.a, c.b)
		}
	}
	if normalizeOutput("1 2") == normalizeOutput("1  2") {
		t.Error("internal whitespace should be significant")
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		out      RunOutput
		expected string
		want     Verdict
	}{
		{"accepted", RunOutput{ExitCode: 0, Stdout: "42\n"}, "42", Accepted},
		{"wrong", RunOutput{ExitCode: 0, Stdout: "41\n"}, "42", WrongAnswer},
		{"timeout beats exit", RunOutput{ExitCode: 137, TimedOut: true}, "x", TimeLimit},
		{"oom beats exit", RunOutput{ExitCode: 137, OOMKilled: true}, "x", MemoryLimit},
		{"output cap", RunOutput{ExitCode: 0, Truncated: true, Stdout: "huge"}, "huge", OutputLimit},
		{"runtime error", RunOutput{ExitCode: 1, Stderr: "boom"}, "x", RuntimeError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classify(tt.out, tt.expected); got != tt.want {
				t.Errorf("classify = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestJudgeAllAccepted(t *testing.T) {
	r := fakeRunner{run: func(spec RunSpec) (RunOutput, error) {
		// Echo the stdin as the answer.
		return RunOutput{ExitCode: 0, Stdout: strings.TrimSpace(spec.Stdin) + "\n"}, nil
	}}
	j := New(r)
	sub := Submission{
		Language: "python",
		Code:     "print(input())",
		Cases: []Case{
			{Name: "1", Input: "hello", Expected: "hello"},
			{Name: "2", Input: "world", Expected: "world"},
		},
	}
	rep, err := j.Judge(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != Accepted || rep.Passed != 2 || rep.Total != 2 {
		t.Fatalf("got %+v", rep)
	}
}

func TestJudgeFirstFailureWins(t *testing.T) {
	r := fakeRunner{run: func(spec RunSpec) (RunOutput, error) {
		if strings.TrimSpace(spec.Stdin) == "bad" {
			return RunOutput{ExitCode: 0, Stdout: "nope\n"}, nil
		}
		return RunOutput{ExitCode: 0, Stdout: strings.TrimSpace(spec.Stdin) + "\n"}, nil
	}}
	j := New(r)
	sub := Submission{
		Language: "python",
		Cases: []Case{
			{Name: "0", Input: "ok", Expected: "ok"},
			{Name: "1", Input: "bad", Expected: "bad"},
			{Name: "2", Input: "ok2", Expected: "ok2"},
		},
	}
	rep, err := j.Judge(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != WrongAnswer {
		t.Fatalf("want WA, got %s", rep.Verdict)
	}
	if rep.Passed != 2 {
		t.Fatalf("want 2 passed, got %d", rep.Passed)
	}
	// Results must be ordered by index regardless of completion order.
	for i, c := range rep.Cases {
		if c.Index != i {
			t.Fatalf("results not ordered: case %d has index %d", i, c.Index)
		}
	}
}

func TestJudgeCompileError(t *testing.T) {
	r := fakeRunner{
		check: func(spec RunSpec) (RunOutput, error) {
			return RunOutput{ExitCode: 1, Stderr: "SyntaxError: bad"}, nil
		},
		run: func(spec RunSpec) (RunOutput, error) {
			t.Fatal("run should not execute when check fails")
			return RunOutput{}, nil
		},
	}
	j := New(r)
	rep, err := j.Judge(context.Background(), Submission{
		Language: "python",
		Code:     "def (",
		Cases:    []Case{{Name: "0", Input: "x", Expected: "x"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != CompileError || rep.Compile == nil {
		t.Fatalf("want CE with compile detail, got %+v", rep)
	}
}

func TestProgressStreaming(t *testing.T) {
	r := fakeRunner{run: func(spec RunSpec) (RunOutput, error) {
		return RunOutput{ExitCode: 0, Stdout: strings.TrimSpace(spec.Stdin) + "\n"}, nil
	}}
	j := New(r)
	sub := Submission{
		Language: "python",
		Cases: []Case{
			{Name: "a", Input: "a", Expected: "a"},
			{Name: "b", Input: "b", Expected: "b"},
			{Name: "c", Input: "c", Expected: "c"},
		},
	}
	progress := make(chan CaseResult, len(sub.Cases))
	go func() {
		_, _ = j.JudgeStream(context.Background(), sub, progress)
		close(progress)
	}()
	count := 0
	for range progress {
		count++
	}
	if count != 3 {
		t.Fatalf("want 3 progress events, got %d", count)
	}
}

func TestCappedBuffer(t *testing.T) {
	c := &cappedBuffer{cap: 10}
	n, _ := c.Write([]byte("12345"))
	if n != 5 {
		t.Fatalf("write should report full length, got %d", n)
	}
	c.Write([]byte("67890EXTRA"))
	if !c.truncated {
		t.Error("expected truncation flag")
	}
	if len(c.String()) != 10 {
		t.Fatalf("expected 10 bytes retained, got %d", len(c.String()))
	}
}

// Guard against an accidental default-limit regression that would neuter the sandbox.
func TestDefaultLimitsAreTight(t *testing.T) {
	l := Limits{}.withDefaults()
	if l.Pids > 512 || l.MemoryMiB > 512 || l.Wall > 30*time.Second {
		t.Fatalf("default limits look too loose: %+v", l)
	}
}
