# marky

[English](README.md) | [Português (Brasil)](README.pt-BR.md)

Render Markdown in the terminal in a clean, pleasant-to-read way. marky wraps text to your window, centers it at a comfortable reading width, and opens an interactive pager when the document is taller than the terminal.

## Install

```sh
go install github.com/silviohfc/marky@latest
```

Or build from a checkout:

```sh
go build -o marky .
```

## Usage

```sh
marky README.md
cat notes.md | marky
marky - < notes.md
marky --style light --width 72 notes.md
marky --no-pager notes.md | less -R
```

With no file, or with `-`, marky reads from standard input.

### Flags

| Flag | Description |
|------|-------------|
| `--style dark\|light\|auto` | Color style. `auto` (default) detects the terminal background. Ignored when stdout is not a TTY. |
| `--width N` | Word-wrap width in columns. Default: terminal width, capped at 100. Without a TTY: 80. |
| `--pager` | Always open the interactive viewer (when stdout is a TTY). |
| `--no-pager` | Always print plainly. |
| `--version` | Print the version. |
| `--help` | Print usage. |

The viewer opens when stdout is a TTY and the rendered document has more lines than the terminal height. When stdout is not a TTY (a pipe or file), marky always prints plainly, without colors or escape codes. The text is still rendered from Markdown (wrapped at 80 columns, with list bullets, table borders and so on), so the output is readable but not byte-identical to the input.

### Pager keys

| Key | Action |
|-----|--------|
| `q`, `Esc`, `Ctrl+C` | Quit |
| `j`, `Down` / `k`, `Up` | Scroll one line |
| `Space`, `f`, `PgDn` / `b`, `PgUp` | Scroll one page |
| `d` / `u` | Scroll half a page |
| `g`, `Home` / `G`, `End` | Top / bottom |
| Mouse wheel | Scroll |

The footer shows the file name, scroll percentage and a key hint. Resizing the terminal re-wraps the document and keeps the scroll position roughly proportional. The pager uses the alternate screen, so your terminal is restored on exit.

## Using marky as a replacement for cat

marky is not a drop-in `cat`: it renders Markdown, so it would mangle other files and does not understand `cat` flags such as `-n`. The safe approach is a wrapper that uses marky only when **every argument is an existing `.md` or `.markdown` file and stdout is a terminal**, and otherwise runs the real `cat` (other files, flags like `-n`, pipes and redirects).

### Safe wrapper

**bash** - `~/.bashrc`

```bash
cat() {
  if [ -t 1 ] && [ "$#" -gt 0 ]; then
    for arg in "$@"; do
      case "$arg" in
        *.md|*.markdown) [ -f "$arg" ] || { command cat "$@"; return; } ;;
        *) command cat "$@"; return ;;
      esac
    done
    command marky "$@"
  else
    command cat "$@"
  fi
}
```

**zsh** - `~/.zshrc`

```zsh
cat() {
  if [ -t 1 ] && [ "$#" -gt 0 ]; then
    for arg in "$@"; do
      case "$arg" in
        *.md|*.markdown) [ -f "$arg" ] || { command cat "$@"; return; } ;;
        *) command cat "$@"; return ;;
      esac
    done
    command marky "$@"
  else
    command cat "$@"
  fi
}
```

**fish** - `~/.config/fish/functions/cat.fish` (or `~/.config/fish/config.fish`)

```fish
function cat --wraps cat --description 'cat, with marky for Markdown files'
    if isatty stdout; and test (count $argv) -gt 0
        for arg in $argv
            if not string match -rq '\.(md|markdown)$' -- $arg; or not test -f $arg
                command cat $argv
                return $status
            end
        end
        command marky $argv
    else
        command cat $argv
    end
end
```

**PowerShell** - `$PROFILE`

```powershell
Remove-Item Alias:cat -Force -ErrorAction SilentlyContinue
function cat {
    $notMarkdown = $args | Where-Object {
        $_ -notmatch '\.(md|markdown)$' -or -not (Test-Path -LiteralPath $_ -PathType Leaf)
    }
    if ($args.Count -gt 0 -and -not $notMarkdown -and -not [Console]::IsOutputRedirected) {
        marky @args
    } else {
        Get-Content @args
    }
}
```

**Nushell** - `config.nu` (`config nu` opens it)

```nu
def --wrapped cat [...args: string] {
    let all_markdown = ($args | is-not-empty) and ($args | all {|a|
        ($a =~ '\.(md|markdown)$') and (($a | path type) == 'file')
    })
    if $all_markdown and (is-terminal --stdout) {
        ^marky ...$args
    } else {
        ^cat ...$args
    }
}
```

**POSIX sh, dash, ksh** - `~/.profile`, or the file named by `ENV` for interactive shells

```sh
cat() {
  if [ -t 1 ] && [ "$#" -gt 0 ]; then
    for arg in "$@"; do
      case "$arg" in
        *.md|*.markdown) [ -f "$arg" ] || { command cat "$@"; return; } ;;
        *) command cat "$@"; return ;;
      esac
    done
    command marky "$@"
  else
    command cat "$@"
  fi
}
```

### The simple alias (not recommended)

An alias is shorter but replaces `cat` unconditionally. `cat -n file`, `cat a.txt b.txt` and `cat file | less` will all go through the Markdown renderer instead of printing the file as-is. Prefer the wrapper above.

Alias variants:

- bash: `alias cat=marky`
- zsh: `alias cat=marky`
- fish: `alias cat marky`
- PowerShell: `Set-Alias -Name cat -Value marky -Option AllScope -Force`
- Nushell: `alias cat = marky`
- POSIX sh, dash, ksh: `alias cat=marky`

### Notes

- Aliases and functions only affect interactive shells. Scripts, `make`, cron and other programs still run the real `cat`.
- Bypass the wrapper for a single call with `command cat file.md` (bash, zsh, fish, POSIX shells) or `\cat file.md` (bash, zsh, POSIX shells). In PowerShell use `Get-Content file.md`; in Nushell use `^cat file.md`.
- Reload your configuration after editing it, or open a new terminal: `source ~/.bashrc`, `source ~/.zshrc`, `. ~/.profile`, `source ~/.config/fish/config.fish`, `. $PROFILE`, or `source $nu.config-path` in Nushell (Nushell loads its config at startup, so restart it).
- The Nushell and PowerShell snippets could not be run on the machine where this was written; they follow each shell's documented syntax, so please report problems.

## Development

```sh
go vet ./...
go test ./...
```

`testdata/sample.md` exercises headings, lists, code, tables, blockquotes, links and rules:

```sh
go run . testdata/sample.md
```

Layout: `main.go` (wiring), `internal/cli` (flags and decisions), `internal/render` (glamour and width), `internal/pager` (bubbletea model).
