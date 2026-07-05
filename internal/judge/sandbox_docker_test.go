package judge

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// requireDocker skips when no Docker daemon is reachable (e.g. `go test -short`
// or a machine without a runtime). In CI, Docker is present and these run.
func requireDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Docker integration test in -short mode")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		t.Skip("Docker not available; skipping sandbox integration test")
	}
}

// judgeSnippet runs a single Python program against one case with tight limits
// and returns the report. Limits are intentionally small to keep hostile tests
// fast and to prove the ceilings bite.
func judgeSnippet(t *testing.T, code string, c Case) Report {
	t.Helper()
	j := New(NewDockerRunner(""))
	sub := Submission{
		Language: "python",
		Code:     code,
		Cases:    []Case{c},
		Limits:   Limits{Wall: 6 * time.Second, MemoryMiB: 128, CPUs: 1, Pids: 64, OutputKiB: 64},
	}
	rep, err := j.Judge(context.Background(), sub)
	if err != nil {
		t.Fatalf("judge error: %v", err)
	}
	return rep
}

// A correct program must pass — proving containment doesn't break normal code.
func TestSandbox_HappyPath(t *testing.T) {
	requireDocker(t)
	rep := judgeSnippet(t, "a,b=map(int,input().split())\nprint(a+b)", Case{Input: "2 3", Expected: "5"})
	if rep.Verdict != Accepted {
		t.Fatalf("want AC, got %s (stderr: %q)", rep.Verdict, firstStderr(rep))
	}
}

func TestSandbox_WrongAnswer(t *testing.T) {
	requireDocker(t)
	rep := judgeSnippet(t, "print(999)", Case{Input: "2 3", Expected: "5"})
	if rep.Verdict != WrongAnswer {
		t.Fatalf("want WA, got %s", rep.Verdict)
	}
}

// Fork bomb: pids-limit must contain it. The only claim that matters is that the
// call RETURNS (host survives) with a non-accepting verdict, bounded in time.
func TestSandbox_ForkBombContained(t *testing.T) {
	requireDocker(t)
	start := time.Now()
	rep := judgeSnippet(t, "import os\nwhile True:\n    os.fork()", Case{Input: "", Expected: "unreachable"})
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Fatalf("fork bomb was not contained promptly: took %s", elapsed)
	}
	if rep.Verdict == Accepted {
		t.Fatalf("fork bomb should never be accepted, got %s", rep.Verdict)
	}
	t.Logf("fork bomb contained with verdict %s", rep.Verdict)
}

// Infinite loop: the wall-clock watchdog must kill it → TLE.
func TestSandbox_InfiniteLoopTLE(t *testing.T) {
	requireDocker(t)
	rep := judgeSnippet(t, "while True:\n    pass", Case{Input: "", Expected: "unreachable"})
	if rep.Verdict != TimeLimit {
		t.Fatalf("want TLE, got %s", rep.Verdict)
	}
}

// Memory bomb: exceeding the cgroup limit must trigger an OOM kill → MLE.
func TestSandbox_MemoryBombMLE(t *testing.T) {
	requireDocker(t)
	code := "a=[]\nwhile True:\n    a.append(bytearray(20*1024*1024))"
	rep := judgeSnippet(t, code, Case{Input: "", Expected: "unreachable"})
	if rep.Verdict != MemoryLimit {
		t.Fatalf("want MLE, got %s", rep.Verdict)
	}
}

// Network egress: --network none must make any outbound connection fail. The
// program crashes rather than printing CONNECTED → RE, and never accepted.
func TestSandbox_NetworkBlocked(t *testing.T) {
	requireDocker(t)
	code := "import socket\ns=socket.socket()\ns.settimeout(4)\ns.connect(('1.1.1.1',53))\nprint('CONNECTED')"
	rep := judgeSnippet(t, code, Case{Input: "", Expected: "CONNECTED"})
	if rep.Verdict == Accepted {
		t.Fatalf("network egress should be blocked, but got AC")
	}
	if strings.Contains(rep.Cases[0].Stdout, "CONNECTED") {
		t.Fatalf("program reached the network")
	}
}

// Read-only root filesystem: writing outside /tmp must fail → RE.
func TestSandbox_ReadOnlyFilesystem(t *testing.T) {
	requireDocker(t)
	code := "open('/pwned','w').write('x')\nprint('WROTE')"
	rep := judgeSnippet(t, code, Case{Input: "", Expected: "WROTE"})
	if rep.Verdict != RuntimeError {
		t.Fatalf("want RE from read-only fs, got %s", rep.Verdict)
	}
	if strings.Contains(rep.Cases[0].Stdout, "WROTE") {
		t.Fatalf("write to root filesystem succeeded")
	}
}

// Real syntax error must surface as CE through the actual check phase.
func TestSandbox_CompileError(t *testing.T) {
	requireDocker(t)
	rep := judgeSnippet(t, "def (oops", Case{Input: "", Expected: "x"})
	if rep.Verdict != CompileError {
		t.Fatalf("want CE, got %s", rep.Verdict)
	}
}

func firstStderr(rep Report) string {
	if len(rep.Cases) > 0 {
		return rep.Cases[0].Stderr
	}
	if rep.Compile != nil {
		return rep.Compile.Stderr
	}
	return ""
}
