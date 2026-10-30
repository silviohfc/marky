package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/silviohfc/marky/internal/pager"
	"github.com/silviohfc/marky/internal/render"
)

// Version overrides the version reported by --version. It can be set with
// -ldflags "-X github.com/silviohfc/marky/internal/cli.Version=v1.2.3";
// otherwise the module version Go stamps into the binary is used.
var Version = ""

func version() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(Version, info, ok)
}

// resolveVersion prefers an explicit version, then the module version from
// the build info (set by go install pkg@vX and by go build in a tagged
// checkout), and falls back to "dev".
func resolveVersion(explicit string, info *debug.BuildInfo, ok bool) string {
	if explicit != "" {
		return explicit
	}
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

var ErrHelp = flag.ErrHelp

const stdinName = "stdin"

type Config struct {
	Style       string
	Width       int
	ForcePager  bool
	ForcePrint  bool
	ShowVersion bool
	Path        string
}

type Env struct {
	Stdin    io.Reader
	StdinTTY bool
	Stdout   io.Writer
	Stderr   io.Writer
	IsTTY    bool
	Size     func() (width, height int)
	Dark     func() bool
	RunPager func(name string, render pager.Renderer) error
}

func Parse(args []string, usageOut io.Writer) (Config, error) {
	cfg := Config{}
	fs := newFlagSet(&cfg)
	positional, err := parseInterspersed(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		printUsage(fs, usageOut)
		return cfg, ErrHelp
	}
	if err != nil {
		return cfg, err
	}
	return cfg, validate(&cfg, positional)
}

func newFlagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("marky", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Style, "style", "auto", "color style: dark, light or auto")
	fs.IntVar(&cfg.Width, "width", 0, "word-wrap width in columns (default: terminal width, max 100)")
	fs.BoolVar(&cfg.ForcePager, "pager", false, "always open the interactive viewer")
	fs.BoolVar(&cfg.ForcePrint, "no-pager", false, "always print, never open the viewer")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")
	return fs
}

func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	positional := []string{}
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

func validate(cfg *Config, positional []string) error {
	switch {
	case cfg.Style != "dark" && cfg.Style != "light" && cfg.Style != "auto":
		return fmt.Errorf("invalid --style %q: want dark, light or auto", cfg.Style)
	case cfg.Width < 0:
		return fmt.Errorf("invalid --width %d: must not be negative", cfg.Width)
	case cfg.ForcePager && cfg.ForcePrint:
		return errors.New("--pager and --no-pager cannot be used together")
	case len(positional) > 1:
		return errors.New("expected at most one input file")
	}
	if len(positional) == 1 {
		cfg.Path = positional[0]
	}
	return nil
}

func printUsage(fs *flag.FlagSet, w io.Writer) {
	fmt.Fprintln(w, "Usage: marky [flags] [FILE.md | -]")
	fmt.Fprintln(w, "Render Markdown in the terminal. Reads piped stdin when FILE is omitted or '-'.")
	fmt.Fprintln(w)
	fs.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(w, "  --%-10s %s\n", f.Name, f.Usage)
	})
}

func ShouldPage(cfg Config, isTTY bool, lines, height int) bool {
	switch {
	case !isTTY, cfg.ForcePrint:
		return false
	case cfg.ForcePager:
		return true
	}
	return lines > height
}

func ReadInput(path string, stdin io.Reader) (name, content string, err error) {
	if path == "" || path == "-" {
		data, err := io.ReadAll(stdin)
		return stdinName, string(data), err
	}
	data, err := os.ReadFile(path)
	return filepath.Base(path), string(data), err
}

func Run(args []string, env Env) int {
	cfg, err := Parse(args, env.Stdout)
	if errors.Is(err, ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "marky: %v\nTry 'marky --help' for usage.\n", err)
		return 2
	}
	if cfg.ShowVersion {
		fmt.Fprintf(env.Stdout, "marky %s\n", version())
		return 0
	}
	if cfg.Path == "" && env.StdinTTY {
		// Nothing piped in: show usage instead of waiting on the keyboard.
		printUsage(newFlagSet(&Config{}), env.Stderr)
		return 2
	}
	name, markdown, err := ReadInput(cfg.Path, env.Stdin)
	if err != nil {
		fmt.Fprintf(env.Stderr, "marky: %v\n", err)
		return 1
	}
	if err := display(cfg, env, name, markdown); err != nil {
		fmt.Fprintf(env.Stderr, "marky: %v\n", err)
		return 1
	}
	return 0
}

func display(cfg Config, env Env, name, markdown string) error {
	style := render.PlainStyle
	if env.IsTTY {
		style = render.ResolveStyle(cfg.Style, env.Dark())
	}
	renderer := func(termWidth int) (string, error) {
		width := render.ContentWidth(cfg.Width, termWidth, env.IsTTY)
		out, err := render.Render(markdown, style, width)
		if err != nil {
			return "", err
		}
		return render.Center(out, termWidth, width), nil
	}

	termWidth, termHeight := env.Size()
	out, err := renderer(termWidth)
	if err != nil {
		return err
	}
	if ShouldPage(cfg, env.IsTTY, countLines(out), termHeight) {
		return env.RunPager(name, renderer)
	}
	_, err = io.WriteString(env.Stdout, out)
	return err
}

func countLines(text string) int {
	return strings.Count(strings.TrimRight(text, "\n"), "\n") + 1
}
