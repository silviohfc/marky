# marky sample document

A short paragraph with **bold**, *italic*, ~~strikethrough~~, `inline code` and a [link](https://github.com/charmbracelet/glamour). This paragraph is long enough to show how marky wraps text to the width of your terminal, capped at a comfortable reading width and centered in the window.

## Lists

- First item
- Second item
  - Nested item
  - Another nested item
- Third item

1. Ordered one
2. Ordered two
3. Ordered three

- [x] Done task
- [ ] Open task

## Code

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, marky!")
}
```

```python
def greet(name: str) -> str:
    return f"Hello, {name}!"
```

## Table

| Flag         | Default | Description                    |
|--------------|---------|--------------------------------|
| `--style`    | auto    | dark, light or auto            |
| `--width`    | 0       | word-wrap width                |
| `--pager`    | false   | always open the viewer         |

## Blockquote

> Simplicity is prerequisite for reliability.
>
> > Nested quotes work too.

---

### Third-level heading

#### Fourth-level heading

##### Fifth-level heading

###### Sixth-level heading

Closing paragraph so the document is tall enough to open the pager in a small terminal.
