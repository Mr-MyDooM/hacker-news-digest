## 2026-05-12 - Accessibility and Interactive Element Fixes
**Learning:** ARIA improvements were good but changing semantic HTML elements (div->button) introduced layout and styling risks. Keep existing container elements unless there's a strong reason to change them; use `role` + `tabindex` when adding interactive semantics to a non-interactive element.
**Action:** Never remove `outline` on `:focus` - use `:focus-visible` for custom focus rings instead. Consolidate CSS rules rather than adding duplicate selectors. `aria-controls` must reference the element being controlled, not a child of it.

## 2026-05-16 - Search Empty State and Accessibility
**Learning:** Interactive empty states should provide immediate feedback via clear messaging and actionable 'Clear' buttons. Using `role="status"` and `aria-live="polite"` ensures screen reader users are notified of dynamic changes without interrupting their flow.
**Action:** Always include a recovery action (like a reset button) in empty states and use appropriate ARIA live regions for status updates.

## 2026-05-18 - Archive Navigation and Visual Cue Persistence
**Learning:** When using JavaScript to dynamically update content within a container that also hosts SVG icons (like the clock icon in post meta), the update logic must target a specific child text element rather than the container's root. This prevents the "flash of unstyled content" where icons disappear after the JS runs. Additionally, for long navigation lists like archives, progressive disclosure ("More..." button) combined with 'e.stopPropagation()' allows users to expand the list without losing their place or closing the menu prematurely.
**Action:** Always wrap dynamic text in a dedicated <span> and use 'e.stopPropagation()' for in-menu toggles.

## 2026-05-21 - Landmark Roles and Atomic UI Reveal
**Learning:** Adding a "Skip to main content" link and the '<main>' landmark significantly improves navigation for keyboard and screen reader users without impacting the visual design. For features calculated asynchronously or client-side (like Reading Time), wrapping the entire UI component (icon + text) in a container that is '.hidden' by default prevents "flash of incomplete UI" and Layout Shift. The component should only be revealed as a single atomic unit once all data is processed.
**Action:** Use landmarks for core sections. For any JS-driven text update, default the parent container to hidden if it contains decorative elements that would otherwise appear "broken" or "empty" before the script runs.
