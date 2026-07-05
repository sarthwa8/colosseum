package judge

import "fmt"

// Language describes how to check and run source for one programming language.
// The judge treats languages as data: adding one is a config entry, not new
// control flow. Check runs once with a writable workdir (syntax check for
// interpreted languages, compilation for compiled ones); Run executes per test
// case with the workdir mounted read-only.
type Language struct {
	Name   string
	Image  string   // docker image, must be pre-pulled
	Source string   // filename the code is written to inside the workdir
	Check  []string // command run once (workdir rw); non-zero exit => CompileError
	Run    []string // command run per case (workdir ro); stdin = case input
}

var languages = map[string]Language{
	"python": {
		Name:   "python",
		Image:  "python:3.12-alpine",
		Source: "solution.py",
		// In-memory compile: detects syntax errors without writing __pycache__,
		// so the check works even though we prefer a read-only-ish workdir.
		Check: []string{"python3", "-c", "compile(open('solution.py').read(), 'solution.py', 'exec')"},
		Run:   []string{"python3", "solution.py"},
	},
	// C++ is wired as data but disabled until the gcc image is validated (M1 is
	// Python-first). Enabling it is uncommenting this block + pulling the image.
	// "cpp": {
	// 	Name:   "cpp",
	// 	Image:  "gcc:14",
	// 	Source: "solution.cpp",
	// 	Check:  []string{"g++", "-O2", "-std=c++20", "-o", "solution", "solution.cpp"},
	// 	Run:    []string{"./solution"},
	// },
}

// LookupLanguage returns the config for a language name.
func LookupLanguage(name string) (Language, error) {
	l, ok := languages[name]
	if !ok {
		return Language{}, fmt.Errorf("unsupported language %q", name)
	}
	return l, nil
}

// SupportedLanguages lists the enabled language names.
func SupportedLanguages() []string {
	out := make([]string, 0, len(languages))
	for name := range languages {
		out = append(out, name)
	}
	return out
}
