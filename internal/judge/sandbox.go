package judge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Runner executes a single command in an isolated environment. Abstracting it
// keeps the judge testable (fakeRunner in tests) and documents the upgrade path:
// swapping DockerRunner for a gVisor/Firecracker-backed Runner changes nothing
// above this interface.
type Runner interface {
	Run(ctx context.Context, spec RunSpec) (RunOutput, error)
}

// RunSpec is one container invocation.
type RunSpec struct {
	Image    string
	Argv     []string
	Stdin    string
	HostDir  string // mounted at /sandbox (holds the source / compiled artifact)
	Writable bool   // mount /sandbox rw (compile/check phase) vs ro (run phase)
	Limits   Limits
}

// RunOutput is the raw result of a container run, before verdict classification.
type RunOutput struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	Duration  time.Duration
	TimedOut  bool // our wall-clock watchdog SIGKILLed it
	OOMKilled bool // the kernel OOM-killer fired (memory limit breach)
	Truncated bool // stdout exceeded the output cap
}

// DockerRunner runs commands as disposable containers via the `docker` CLI.
// Shelling out (rather than the SDK) keeps every security flag visible and
// auditable — which is the point of the exercise.
type DockerRunner struct {
	Docker string // path/name of the docker binary; defaults to "docker"
}

// NewDockerRunner returns a runner using the given docker binary ("" => "docker").
func NewDockerRunner(dockerBin string) *DockerRunner {
	if dockerBin == "" {
		dockerBin = "docker"
	}
	return &DockerRunner{Docker: dockerBin}
}

// securityArgs returns the containment flags applied to every run. Each is
// load-bearing; see docs/SANDBOX.md for what attack each one defeats.
func (r *DockerRunner) securityArgs(name string, spec RunSpec) []string {
	lim := spec.Limits.withDefaults()
	mem := strconv.FormatInt(lim.MemoryMiB, 10) + "m"
	mountMode := "ro"
	if spec.Writable {
		mountMode = "rw"
	}
	return []string{
		"run", "-i", "--name", name,
		"--network", "none", //          no network: blocks exfiltration & C2
		"--memory", mem, //              hard RAM ceiling
		"--memory-swap", mem, //         equal => swap disabled => real OOM kill on breach
		"--cpus", strconv.FormatFloat(lim.CPUs, 'f', -1, 64), // CPU quota
		"--pids-limit", strconv.FormatInt(lim.Pids, 10), //     caps threads/procs => fork-bomb containment
		"--read-only", //                immutable root filesystem
		"--tmpfs", "/tmp:rw,size=64m,noexec,nosuid,nodev", // small scratch, no exec of dropped binaries
		"--workdir", "/sandbox",
		"-v", spec.HostDir + ":/sandbox:" + mountMode,
		"--cap-drop", "ALL", //          drop every Linux capability
		"--security-opt", "no-new-privileges", // no setuid escalation
		"--user", "65534:65534", //      run as nobody, never root
	}
}

func randName() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "colosseum-" + hex.EncodeToString(b)
}

// Run executes spec and returns the raw output. It never trusts the container
// to terminate on its own: an independent watchdog SIGKILLs it at the wall
// limit, and the container is force-removed on every path.
func (r *DockerRunner) Run(ctx context.Context, spec RunSpec) (RunOutput, error) {
	lim := spec.Limits.withDefaults()
	name := randName()

	args := r.securityArgs(name, spec)
	args = append(args, spec.Image)
	args = append(args, spec.Argv...)

	// Deliberately NOT exec.CommandContext: cancelling ctx would kill the docker
	// client and orphan the container. We manage the container's lifetime by name.
	cmd := exec.Command(r.Docker, args...)
	cmd.Stdin = strings.NewReader(spec.Stdin)

	outCap := &cappedBuffer{cap: lim.OutputKiB * 1024}
	errCap := &cappedBuffer{cap: 64 * 1024}
	cmd.Stdout = outCap
	cmd.Stderr = errCap

	// Always force-remove the container, even if docker run itself errored.
	defer func() {
		rm := exec.Command(r.Docker, "rm", "-f", name)
		_ = rm.Run()
	}()

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return RunOutput{}, fmt.Errorf("starting docker: %w", err)
	}

	var (
		timedOut bool
		killMu   sync.Mutex
	)
	kill := func() {
		killMu.Lock()
		defer killMu.Unlock()
		_ = exec.Command(r.Docker, "kill", "--signal=KILL", name).Run()
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return r.finish(ctx, name, outCap, errCap, start, err, false)
	case <-time.After(lim.Wall):
		timedOut = true
		kill()
		<-done // reap the docker client now that the container is dying
		return r.finish(ctx, name, outCap, errCap, start, nil, timedOut)
	case <-ctx.Done():
		kill()
		<-done
		out, _ := r.finish(ctx, name, outCap, errCap, start, nil, false)
		return out, ctx.Err()
	}
}

// finish assembles the RunOutput, reading the container's true exit code and
// OOM flag via docker inspect (more reliable than the docker-run exit code,
// which is 137 for both OOM and our own kill).
func (r *DockerRunner) finish(ctx context.Context, name string, out, errb *cappedBuffer, start time.Time, waitErr error, timedOut bool) (RunOutput, error) {
	dur := time.Since(start)
	exit, oom := r.inspect(name)
	if exit < 0 {
		// inspect failed; fall back to the wait error's exit code.
		if ee, ok := waitErr.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else if waitErr == nil {
			exit = 0
		} else {
			return RunOutput{}, fmt.Errorf("docker run: %w", waitErr)
		}
	}
	return RunOutput{
		Stdout:    out.String(),
		Stderr:    errb.String(),
		ExitCode:  exit,
		Duration:  dur,
		TimedOut:  timedOut,
		OOMKilled: oom,
		Truncated: out.truncated,
	}, nil
}

// inspect returns (exitCode, oomKilled). exitCode is -1 if inspection failed.
func (r *DockerRunner) inspect(name string) (int, bool) {
	cmd := exec.Command(r.Docker, "inspect", "--format", "{{.State.ExitCode}} {{.State.OOMKilled}}", name)
	b, err := cmd.Output()
	if err != nil {
		return -1, false
	}
	fields := strings.Fields(strings.TrimSpace(string(b)))
	if len(fields) != 2 {
		return -1, false
	}
	code, err := strconv.Atoi(fields[0])
	if err != nil {
		return -1, false
	}
	return code, fields[1] == "true"
}

// cappedBuffer stores at most cap bytes but keeps accepting writes, so the
// container's stdout pipe never blocks (which would stall until the watchdog).
type cappedBuffer struct {
	cap       int64
	buf       strings.Builder
	n         int64
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if c.n < c.cap {
		room := c.cap - c.n
		if int64(len(p)) <= room {
			c.buf.Write(p)
			c.n += int64(len(p))
		} else {
			c.buf.Write(p[:room])
			c.n = c.cap
			c.truncated = true
		}
	} else {
		c.truncated = true
	}
	return len(p), nil // always fully consume => never backpressure the container
}

func (c *cappedBuffer) String() string { return c.buf.String() }
