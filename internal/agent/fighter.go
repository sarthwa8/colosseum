package agent

import (
	"context"
	"regexp"
	"strings"
)

// Config parameterizes one fighter (competitor) in a match.
type Config struct {
	ID          string   // display id, e.g. "A" or a model nickname
	Provider    Provider // where completions come from
	Model       string   // model id string
	Temperature float64  // advisory; not all providers honor it
	MaxTokens   int      // per-completion cap
}

// Session is a single fighter's conversation for one role in a match. It tracks
// the running transcript and cumulative token usage so the match layer can
// enforce budgets and report cost.
type Session struct {
	cfg      Config
	system   string
	messages []Message
	Usage    Usage
	Turns    int
}

// NewSession starts a conversation with a system prompt.
func NewSession(cfg Config, system string) *Session {
	return &Session{cfg: cfg, system: system}
}

// Config exposes the fighter config (model, id) for manifests and cost.
func (s *Session) Config() Config { return s.cfg }

// Send appends a user turn, gets a completion, appends the assistant reply, and
// returns the raw reply text. Usage accumulates across the session.
func (s *Session) Send(ctx context.Context, user string) (string, error) {
	s.messages = append(s.messages, Message{Role: User, Content: user})
	resp, err := s.cfg.Provider.Complete(ctx, Request{
		System:      s.system,
		Messages:    s.messages,
		Model:       s.cfg.Model,
		Temperature: s.cfg.Temperature,
		MaxTokens:   s.cfg.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	s.messages = append(s.messages, Message{Role: Assistant, Content: resp.Text})
	s.Usage.InputTokens += resp.Usage.InputTokens
	s.Usage.OutputTokens += resp.Usage.OutputTokens
	s.Turns++
	return resp.Text, nil
}

var fenceRe = regexp.MustCompile("(?s)```[a-zA-Z0-9_+-]*\\n(.*?)```")

// ExtractCode returns the last fenced code block in text (models sometimes
// explain, then give the final program). If no fence is present, the whole
// trimmed text is returned as a best effort.
func ExtractCode(text string) string {
	matches := fenceRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(matches[len(matches)-1][1])
}

// SolveSystemPrompt is the fighter's persona for the solve role.
func SolveSystemPrompt(lang string) string {
	return "You are an elite competitive programmer. Solve the problem in " + lang + ".\n" +
		"Read all input from standard input and write the answer to standard output.\n" +
		"Respond with exactly one fenced code block containing the complete, runnable program. " +
		"Do not include explanations outside the code block."
}

// SolvePrompt is the initial user turn: statement + constraints.
func SolvePrompt(statement, constraints string) string {
	var b strings.Builder
	b.WriteString(statement)
	if constraints != "" {
		b.WriteString("\n\n## Input Constraints\n")
		b.WriteString(constraints)
	}
	b.WriteString("\n\nWrite the complete solution now.")
	return b.String()
}

// FeedbackPrompt tells the fighter its submission failed, without leaking hidden
// expected outputs. For crashes/compile errors the stderr is included (that's
// the fighter's own program output); for wrong answers only the pass count is
// revealed — mirroring how real judges report "Wrong Answer on test N".
func FeedbackPrompt(verdict string, passed, total int, stderr string) string {
	var b strings.Builder
	b.WriteString("Your submission was not accepted.\n")
	b.WriteString("Verdict: " + verdict + " — passed " + itoa(passed) + "/" + itoa(total) + " hidden tests.\n")
	if stderr = strings.TrimSpace(stderr); stderr != "" {
		b.WriteString("\nProgram error output:\n")
		b.WriteString(truncate(stderr, 800))
		b.WriteString("\n")
	}
	b.WriteString("\nFind the bug and respond with a corrected complete program in one fenced code block.")
	return b.String()
}

// AttackSystemPrompt is the adversary persona for the Attack/Defense format.
func AttackSystemPrompt() string {
	return "You are a meticulous adversarial tester. You are given a programming problem, " +
		"its input constraints, and another program that claims to solve it. Find a SINGLE input, " +
		"valid under the stated constraints, on which the program produces the WRONG answer, crashes, " +
		"or times out — an edge case the author likely missed (empty/extreme values, overflow, " +
		"boundary sizes, ties, negatives, etc.).\n" +
		"Respond with ONLY that input, exactly as it should be fed on standard input, inside one " +
		"fenced code block. No explanation, no commentary."
}

// AttackPrompt presents the problem and the defender's program to break.
func AttackPrompt(statement, constraints, defenderCode string) string {
	var b strings.Builder
	b.WriteString(statement)
	if constraints != "" {
		b.WriteString("\n\n## Input Constraints (your input MUST satisfy these)\n")
		b.WriteString(constraints)
	}
	b.WriteString("\n\n## Program under test\n```python\n")
	b.WriteString(defenderCode)
	b.WriteString("\n```\n\nProvide one breaking input in a single fenced code block.")
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
