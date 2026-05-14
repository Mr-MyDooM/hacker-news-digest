## 2026-05-12 - Accessibility and Interactive Element Fixes
**Learning:** ARIA improvements were good but changing semantic HTML elements (div->button) introduced layout and styling risks. Keep existing container elements unless there's a strong reason to change them; use `role` + `tabindex` when adding interactive semantics to a non-interactive element.
**Action:** Never remove `outline` on `:focus` - use `:focus-visible` for custom focus rings instead. Consolidate CSS rules rather than adding duplicate selectors. `aria-controls` must reference the element being controlled, not a child of it.

## 2026-05-14 - Interactive Empty States and Utility Classes
**Learning:** Empty states should be interactive and accessible. Using a "Clear search" button in a "No results" message improves recoverability. Relying on implicit utility classes (like `.hidden`) can be risky if they aren't explicitly defined in the project's CSS, potentially leading to visual regressions.
**Action:** Always verify that utility classes used for state management (like `.hidden`) are explicitly defined and use `!important` to ensure they override other display properties. Use `role="status"` and `aria-live="polite"` for dynamic content areas like empty search results.
