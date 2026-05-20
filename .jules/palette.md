## 2026-05-12 - Accessibility and Interactive Element Fixes
**Learning:** ARIA improvements were good but changing semantic HTML elements (div->button) introduced layout and styling risks. Keep existing container elements unless there's a strong reason to change them; use `role` + `tabindex` when adding interactive semantics to a non-interactive element.
**Action:** Never remove `outline` on `:focus` - use `:focus-visible` for custom focus rings instead. Consolidate CSS rules rather than adding duplicate selectors. `aria-controls` must reference the element being controlled, not a child of it.

## 2026-05-16 - Search Empty State and Accessibility
**Learning:** Interactive empty states should provide immediate feedback via clear messaging and actionable 'Clear' buttons. Using `role="status"` and `aria-live="polite"` ensures screen reader users are notified of dynamic changes without interrupting their flow.
**Action:** Always include a recovery action (like a reset button) in empty states and use appropriate ARIA live regions for status updates.

## 2026-05-18 - Archive Navigation and Visual Cue Persistence
**Learning:** When using JavaScript to dynamically update content within a container that also hosts SVG icons (like the clock icon in post meta), the update logic must target a specific child text element rather than the container's root. This prevents the "flash of unstyled content" where icons disappear after the JS runs. Additionally, for long navigation lists like archives, progressive disclosure ("More..." button) combined with 'e.stopPropagation()' allows users to expand the list without losing their place or closing the menu prematurely.
**Action:** Always wrap dynamic text in a dedicated <span> and use 'e.stopPropagation()' for in-menu toggles.

## 2026-05-20 - Transient Feedback for Clipboard Actions
**Learning:** For actions with no permanent UI change (like "Copy to Clipboard"), providing immediate textual feedback via tooltips (e.g., "Copied!") is significantly more discoverable and reassuring than subtle icon-only animations. Integrating these with Bootstrap's tooltip system allows for a consistent look and feel across the application while providing a clear "success" state for an otherwise invisible operation.
**Action:** Use transient tooltips to confirm successful clipboard operations and provide descriptive labels for all icon-only buttons, including keyboard shortcut hints where applicable.
