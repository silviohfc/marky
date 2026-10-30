package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/silviohfc/marky/internal/pager"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]string{}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Style != "auto" || cfg.Width != 0 || cfg.Path != "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestParseReadsFlagsAndPath(t *testing.T) {
	cfg, err := Parse([]string{"--style", "light", "--width", "50", "--pager", "doc.md"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Style != "light" || cfg.Width != 50 || !cfg.ForcePager || cfg.Path != "doc.md" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestParseAcceptsFlagsAfterPath(t *testing.T) {
	cfg, err := Parse([]string{"doc.md", "--no-pager", "--width", "40"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Path != "doc.md" || !cfg.ForcePrint || cfg.Width != 40 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestParseTreatsDashAsStdin(t *testing.T) {
	cfg, err := Parse([]string{"-"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Path != "-" {
		t.Fatalf("got %q", cfg.Path)
	}
}

func TestParseRejectsConflictingPagerFlags(t *testing.T) {
	if _, err := Parse([]string{"--pager", "--no-pager"}, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsUnknownStyle(t *testing.T) {
	if _, err := Parse([]string{"--style", "neon"}, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsNegativeWidth(t *testing.T) {
	if _, err := Parse([]string{"--width", "-3"}, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseRejectsMultipleFiles(t *testing.T) {
	if _, err := Parse([]string{"a.md", "b.md"}, io.Discard); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseReportsHelpRequest(t *testing.T) {
	var out bytes.Buffer
	_, err := Parse([]string{"--help"}, &out)
	if !errors.Is(err, ErrHelp) {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(out.String(), "--pager") {
		t.Fatalf("usage missing flags: %q", out.String())
	}
}

func TestShouldPageWhenTallerThanTerminal(t *testing.T) {
	if !ShouldPage(Config{}, true, 50, 24) {
		t.Fatal("expected pager")
	}
}

func TestShouldNotPageWhenContentFits(t *testing.T) {
	if ShouldPage(Config{}, true, 24, 24) {
		t.Fatal("expected plain print")
	}
}

func TestShouldNotPageWithoutTTY(t *testing.T) {
	if ShouldPage(Config{}, false, 500, 24) {
		t.Fatal("expected plain print")
	}
}

func TestForcePagerOpensPagerForShortContent(t *testing.T) {
	if !ShouldPage(Config{ForcePager: true}, true, 2, 24) {
		t.Fatal("expected pager")
	}
}

func TestForcePagerStillPrintsWithoutTTY(t *testing.T) {
	if ShouldPage(Config{ForcePager: true}, false, 2, 24) {
		t.Fatal("expected plain print")
	}
}

func TestForcePrintSkipsPagerForTallContent(t *testing.T) {
	if ShouldPage(Config{ForcePrint: true}, true, 500, 24) {
		t.Fatal("expected plain print")
	}
}

func TestReadInputReadsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	name, content, err := ReadInput(path, strings.NewReader("ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "doc.md" || content != "# hi" {
		t.Fatalf("got %q %q", name, content)
	}
}

func TestReadInputReadsStdinWhenPathEmpty(t *testing.T) {
	name, content, err := ReadInput("", strings.NewReader("from stdin"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "stdin" || content != "from stdin" {
		t.Fatalf("got %q %q", name, content)
	}
}

func TestReadInputReadsStdinWhenPathIsDash(t *testing.T) {
	name, content, err := ReadInput("-", strings.NewReader("piped"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "stdin" || content != "piped" {
		t.Fatalf("got %q %q", name, content)
	}
}

func TestReadInputFailsForMissingFile(t *testing.T) {
	if _, _, err := ReadInput("/nonexistent/nope.md", strings.NewReader("")); err == nil {
		t.Fatal("expected error")
	}
}

func newEnv(stdin string, tty bool, w, h int) (Env, *bytes.Buffer, *bytes.Buffer, *[]string) {
	var out, errOut bytes.Buffer
	opened := []string{}
	env := Env{
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Stderr: &errOut,
		IsTTY:  tty,
		Size:   func() (int, int) { return w, h },
		Dark:   func() bool { return true },
		RunPager: func(name string, r pager.Renderer) error {
			opened = append(opened, name)
			return nil
		},
	}
	return env, &out, &errOut, &opened
}

func TestRunPrintsRenderedMarkdownWithoutTTY(t *testing.T) {
	env, out, _, opened := newEnv("# Title\n\nbody", false, 0, 0)
	code := Run([]string{}, env)
	if code != 0 || !strings.Contains(out.String(), "Title") || len(*opened) != 0 {
		t.Fatalf("code=%d out=%q opened=%v", code, out.String(), *opened)
	}
}

func TestRunOpensPagerWhenContentIsTallerThanTerminal(t *testing.T) {
	env, out, _, opened := newEnv(strings.Repeat("para\n\n", 40), true, 100, 10)
	code := Run([]string{}, env)
	if code != 0 || out.Len() != 0 || len(*opened) != 1 || (*opened)[0] != "stdin" {
		t.Fatalf("code=%d out=%q opened=%v", code, out.String(), *opened)
	}
}

func TestRunReportsMissingFileOnStderr(t *testing.T) {
	env, _, errOut, _ := newEnv("", false, 0, 0)
	code := Run([]string{"/nonexistent/nope.md"}, env)
	if code == 0 || !strings.Contains(errOut.String(), "nope.md") {
		t.Fatalf("code=%d err=%q", code, errOut.String())
	}
}

func TestRunReportsUsageErrorsWithExitCode2(t *testing.T) {
	env, _, errOut, _ := newEnv("", false, 0, 0)
	code := Run([]string{"--style", "neon"}, env)
	if code != 2 || errOut.Len() == 0 {
		t.Fatalf("code=%d err=%q", code, errOut.String())
	}
}

func TestRunHelpExitsZero(t *testing.T) {
	env, out, _, _ := newEnv("", false, 0, 0)
	if code := Run([]string{"--help"}, env); code != 0 || !strings.Contains(out.String(), "Usage") {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestRunVersionPrintsVersion(t *testing.T) {
	env, out, _, _ := newEnv("", false, 0, 0)
	if code := Run([]string{"--version"}, env); code != 0 || !strings.Contains(out.String(), version()) {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestRunPrintsPlainTextWithoutTTY(t *testing.T) {
	env, out, _, _ := newEnv("# Title\n\n**bold**", false, 0, 0)
	Run([]string{}, env)
	if strings.Contains(out.String(), "\x1b") {
		t.Fatalf("unexpected escape codes: %q", out.String())
	}
}

func TestRunShowsUsageInsteadOfWaitingWhenStdinIsTerminal(t *testing.T) {
	env, out, errOut, opened := newEnv("", true, 100, 40)
	env.StdinTTY = true
	code := Run([]string{}, env)
	if code != 2 || !strings.Contains(errOut.String(), "Usage") || out.Len() != 0 || len(*opened) != 0 {
		t.Fatalf("code=%d out=%q err=%q opened=%v", code, out.String(), errOut.String(), *opened)
	}
}

func TestRunReadsTerminalStdinWhenDashIsExplicit(t *testing.T) {
	env, out, _, _ := newEnv("# Typed", false, 0, 0)
	env.StdinTTY = true
	if code := Run([]string{"-"}, env); code != 0 || !strings.Contains(out.String(), "Typed") {
		t.Fatalf("code=%d out=%q", code, out.String())
	}
}

func TestResolveVersionPrefersExplicitValue(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.3.0"}}
	if got := resolveVersion("v9.9.9", info, true); got != "v9.9.9" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveVersionUsesModuleVersionFromBuildInfo(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.3.1"}}
	if got := resolveVersion("", info, true); got != "v0.3.1" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveVersionFallsBackToDev(t *testing.T) {
	cases := map[string]*debug.BuildInfo{
		"devel":   {Main: debug.Module{Version: "(devel)"}},
		"empty":   {},
		"no info": nil,
	}
	for name, info := range cases {
		if got := resolveVersion("", info, info != nil); got != "dev" {
			t.Errorf("%s: got %q", name, got)
		}
	}
}
