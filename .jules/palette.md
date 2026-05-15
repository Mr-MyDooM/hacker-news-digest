## 2026-05-12 - Accessibility and Interactive Element Fixes
**Learning:** ARIA improvements were good but changing semantic HTML elements (div->button) introduced layout and styling risks. Keep existing container elements unless there's a strong reason to change them; use `role` + `tabindex` when adding interactive semantics to a non-interactive element.
**Action:** Never remove `outline` on `:focus` - use `:focus-visible` for custom focus rings instead. Consolidate CSS rules rather than adding duplicate selectors. `aria-controls` must reference the element being controlled, not a child of it.

## 2026-05-15 - Search Empty States and Recoverability
**Learning:** Providing a "No results" message without a way to reset the state forces users to manually delete their query, increasing friction. An interactive empty state with a "Clear" button improves recoverability.
**Action:** Always pair dynamic empty states with an actionable reset component. Use `role="status"` and `aria-live="polite"` to ensure the state change is announced to assistive technologies.
