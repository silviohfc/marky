package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
