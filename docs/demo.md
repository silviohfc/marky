# Release notes

Version **2.0** brings a *faster* parser, a new `--watch` flag and a
cleaner theme. Read the full guide at [marky docs](https://github.com/silviohfc/marky).

## Highlights

- Instant rendering, even for huge files
- Syntax highlighting for 200+ languages
- [x] Dark and light themes
- [ ] Live reload

## Quick start

```go
func main() {
	doc, _ := os.ReadFile("README.md")
	fmt.Println(marky.Render(doc))
}
```

| Command      | What it does       |
|--------------|--------------------|
| `marky a.md` | Render a file      |
| `marky -`    | Read from stdin    |

> Simplicity is prerequisite for reliability.
