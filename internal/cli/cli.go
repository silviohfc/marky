package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var Version = "dev"

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

func Parse(args []string, usageOut io.Writer) (Config, error) {
	cfg := Config{}
	fs := flag.NewFlagSet("marky", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Style, "style", "auto", "color style: dark, light or auto")
	fs.IntVar(&cfg.Width, "width", 0, "word-wrap width in columns (default: terminal width, max 100)")
	fs.BoolVar(&cfg.ForcePager, "pager", false, "always open the interactive viewer")
	fs.BoolVar(&cfg.ForcePrint, "no-pager", false, "always print, never open the viewer")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")

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
	fmt.Fprintln(w, "Render Markdown in the terminal. Reads stdin when FILE is omitted or '-'.")
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
