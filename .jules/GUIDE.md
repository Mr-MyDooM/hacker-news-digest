# Jules Code Review Guide

## DO NOT

### Accessibility
1. **Never remove `outline` on `:focus`** — this breaks keyboard navigation. Use `:focus-visible` for custom focus rings.
2. **Don't swap semantic HTML elements** (div->button, a->button) unless layout impact is verified. Use `role="button"` + `tabindex="0"` on existing elements instead.
3. **`aria-controls` must point to the controlled element**, not a child inside it. Verify the ID target exists in the template.
4. **Don't duplicate CSS selectors** — consolidate `:focus-within`, `:hover` groups, etc. into one block. Duplicate blocks override each other unpredictably.

### Performance / Architecture
5. **Don't duplicate cache/business logic** across functions. Extract shared check into a helper, or let one function own it with fallback in the other.
6. **Check that early-return paths don't skip essential side effects** (e.g. image extraction, cache writes). If a cached entry is incomplete (missing image), a later pass should retry, not skip.

### PR Hygiene
7. **Don't include learning/doc file diffs in the same PR as code changes** — commit them separately or leave them out.
8. **Verify your branch compiles before pushing** (`go build ./...`, `make build`).

## DO

1. Use `io.LimitReader` on all external HTTP reads — this is always correct.
2. Add ARIA attributes (`aria-expanded`, `aria-controls`, `aria-label`) — they improve accessibility when the target IDs are correct.
3. Use semantic `<button>` for clickable actions that don't navigate — this is correct when CSS is fully accounted for.
4. Check for `err` shadowing when introducing new variables in existing scopes.
