## 2026-05-12 - Accessibility and Interactive Element Fixes
**Learning:** ARIA improvements were good but changing semantic HTML elements (div->button) introduced layout and styling risks. Keep existing container elements unless there's a strong reason to change them; use `role` + `tabindex` when adding interactive semantics to a non-interactive element.
**Action:** Never remove `outline` on `:focus` - use `:focus-visible` for custom focus rings instead. Consolidate CSS rules rather than adding duplicate selectors. `aria-controls` must reference the element being controlled, not a child of it.

## 2026-05-16 - Search Empty State and Accessibility
**Learning:** Interactive empty states should provide immediate feedback via clear messaging and actionable 'Clear' buttons. Using `role="status"` and `aria-live="polite"` ensures screen reader users are notified of dynamic changes without interrupting their flow.
**Action:** Always include a recovery action (like a reset button) in empty states and use appropriate ARIA live regions for status updates.
