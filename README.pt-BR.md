# marky

[English](README.md) | [Português (Brasil)](README.pt-BR.md)

Renderiza Markdown no terminal de forma limpa e agradável de ler. O marky quebra o texto conforme a largura da janela, centraliza em uma largura de leitura confortável e abre um pager interativo quando o documento é mais alto que o terminal.

## Instalação

```sh
go install github.com/silviohfc/marky@latest
```

Ou compile a partir do código-fonte:

```sh
go build -o marky .
```

## Uso

```sh
marky README.md
cat notes.md | marky
marky - < notes.md
marky --style light --width 72 notes.md
marky --no-pager notes.md | less -R
```

Sem arquivo, ou com `-`, o marky lê da entrada padrão.

### Opções

| Opção | Descrição |
|------|-------------|
| `--style dark\|light\|auto` | Estilo de cores. `auto` (padrão) detecta o fundo do terminal. Ignorado quando a saída não é um TTY. |
| `--width N` | Largura de quebra de linha em colunas. Padrão: largura do terminal, limitada a 100. Sem TTY: 80. |
| `--pager` | Sempre abre o visualizador interativo (quando a saída é um TTY). |
| `--no-pager` | Sempre imprime diretamente. |
| `--version` | Mostra a versão. |
| `--help` | Mostra a ajuda. |

O visualizador abre quando a saída é um TTY e o documento renderizado tem mais linhas que a altura do terminal. Quando a saída não é um TTY (pipe ou arquivo), o marky sempre imprime diretamente, sem cores nem códigos de escape. O texto continua sendo renderizado a partir do Markdown (quebrado em 80 colunas, com marcadores de lista, bordas de tabela etc.), então a saída é legível, mas não é idêntica byte a byte à entrada.

### Teclas do pager

| Tecla | Ação |
|-----|--------|
| `q`, `Esc`, `Ctrl+C` | Sair |
| `j`, `Down` / `k`, `Up` | Rolar uma linha |
| `Space`, `f`, `PgDn` / `b`, `PgUp` | Rolar uma página |
| `d` / `u` | Rolar meia página |
| `g`, `Home` / `G`, `End` | Topo / fim |
| Roda do mouse | Rolar |

O rodapé mostra o nome do arquivo, a porcentagem de rolagem e uma dica de teclas. Ao redimensionar o terminal, o documento é requebrado e a posição de rolagem é mantida aproximadamente proporcional. O pager usa a tela alternativa, então o terminal é restaurado ao sair.

## Usando o marky no lugar do cat

O marky não é um substituto direto do `cat`: ele renderiza Markdown, então deformaria outros arquivos e não entende opções do `cat` como `-n`. O jeito seguro é uma função que usa o marky apenas quando **todos os argumentos são arquivos `.md` ou `.markdown` existentes e a saída é um terminal**, e caso contrário executa o `cat` de verdade (outros arquivos, opções como `-n`, pipes e redirecionamentos).

### Função segura

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

**fish** - `~/.config/fish/functions/cat.fish` (ou `~/.config/fish/config.fish`)

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

**Nushell** - `config.nu` (`config nu` abre o arquivo)

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

**POSIX sh, dash, ksh** - `~/.profile`, ou o arquivo apontado por `ENV` em shells interativos

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

### O alias simples (não recomendado)

Um alias é mais curto, mas substitui o `cat` incondicionalmente. `cat -n arquivo`, `cat a.txt b.txt` e `cat arquivo | less` passarão pelo renderizador de Markdown em vez de imprimir o arquivo como está. Prefira a função acima.

Variantes com alias:

- bash: `alias cat=marky`
- zsh: `alias cat=marky`
- fish: `alias cat marky`
- PowerShell: `Set-Alias -Name cat -Value marky -Option AllScope -Force`
- Nushell: `alias cat = marky`
- POSIX sh, dash, ksh: `alias cat=marky`

### Observações

- Aliases e funções afetam apenas shells interativos. Scripts, `make`, cron e outros programas continuam usando o `cat` de verdade.
- Para ignorar a função em uma única chamada, use `command cat arquivo.md` (bash, zsh, fish, shells POSIX) ou `\cat arquivo.md` (bash, zsh, shells POSIX). No PowerShell use `Get-Content arquivo.md`; no Nushell use `^cat arquivo.md`.
- Recarregue a configuração depois de editá-la, ou abra um novo terminal: `source ~/.bashrc`, `source ~/.zshrc`, `. ~/.profile`, `source ~/.config/fish/config.fish`, `. $PROFILE`; no Nushell a configuração é lida na inicialização, então reinicie-o.
- Os trechos de Nushell e PowerShell não puderam ser executados na máquina onde isto foi escrito; eles seguem a sintaxe documentada de cada shell, então avise se algo falhar.

## Desenvolvimento

```sh
go vet ./...
go test ./...
```

`testdata/sample.md` exercita títulos, listas, código, tabelas, citações, links e linhas horizontais:

```sh
go run . testdata/sample.md
```

Estrutura: `main.go` (montagem), `internal/cli` (opções e decisões), `internal/render` (glamour e largura), `internal/pager` (modelo bubbletea).
