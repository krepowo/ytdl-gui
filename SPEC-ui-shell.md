# Spec: ui-shell

## Objective
Provide the app's visual frame and design system: a frameless window with a
custom dark title bar, and the coral-on-charcoal theme expressed once as tokens
shared by **MUI v9** and **Tailwind v4**. The UI is **icon-first**: prefer
Material icons with tooltips over text labels; use text only where an icon alone
would be ambiguous (e.g. the primary "Download" button).

## Tech Stack (pinned to latest)
| Package | Version |
|---|---|
| react / react-dom | 19.3.x |
| @mui/material | 9.4.x |
| @mui/icons-material | 9.4.x |
| @emotion/react / @emotion/styled | 11.14.x |
| tailwindcss + @tailwindcss/vite | 4.3.x |
| vite | 8.3.x |
| @vitejs/plugin-react | 6.1.x |
| typescript | **5.x** (pinned; the 7.x native compiler is not used — plugin compatibility) |
| vitest | 5.0.x |

## Commands
- Dev: `wails dev`
- Build: `wails build`
- Lint: `npm run lint`
- Typecheck: `npm run typecheck`

## Project Structure
```
frontend/
  src/
    theme/
      tokens.ts        → single source of truth for colors/radius/space
      muiTheme.ts      → MUI v9 theme built from tokens
    components/
      TitleBar.tsx     → custom dark title bar (drag, min, close)
      Layout.tsx       → window padding, page container
    App.tsx
  src/index.css        → @import "tailwindcss"; @theme { ...tokens... }
  vite.config.ts       → react() + tailwindcss() plugins
```

## Tailwind v4 (CSS-first — no config file)
```css
/* src/index.css */
@import "tailwindcss";

/* Map the same design tokens to Tailwind utilities. */
@theme {
  --color-app-bg:      #0a0a0c;
  --color-window-bg:   #121214;
  --color-surface:     #18181b;
  --color-surface-alt: #1f1f24;
  --color-titlebar:    #232329;
  --color-border:      #27272a;
  --color-primary:     #ef5350;
  --color-primary-hi:  #f87171;
  --color-success:     #34d399;
  --color-text:        #f3f4f6;
  --color-text-muted:  #9ca3af;
  --radius-window: 18px;
  --radius-control: 12px;
  --radius-card: 14px;
}
```
- **Do NOT create `tailwind.config.js` or `postcss.config.js`** — v4 is CSS-first.
- `vite.config.ts` uses the `@tailwindcss/vite` plugin (no PostCSS).

## MUI v9 Integration
- Wrap the app in `ThemeProvider` with a theme built from `tokens.ts`; override
  `palette.background`, `palette.primary`, and `shape.borderRadius`.
- Enable CSS variables mode (MUI v9 emits `color-mix()`-derived colors) so
  Tailwind utilities and MUI components resolve the same values.
- **Preflight vs CssBaseline:** import Tailwind first, then mount MUI's
  `CssBaseline`. Verify button/input resets don't fight (check `button` background
  and heading margins) and add `@layer base` overrides only if needed.

## Code Style
```ts
// tokens.ts — the ONLY place raw hex values live.
export const tokens = {
  color: {
    appBg: "#0a0a0c", windowBg: "#121214", surface: "#18181b",
    surfaceAlt: "#1f1f24", titleBar: "#232329", border: "#27272a",
    primary: "#ef5350", primaryHi: "#f87171", success: "#34d399",
    text: "#f3f4f6", textMuted: "#9ca3af",
  },
  radius: { window: 18, control: 12, card: 14 },
  space:  { page: 24, gap: 16 },
} as const;
```
```tsx
// Icon-first: icon button + tooltip instead of a text label.
<Tooltip title="Buka Folder">
  <IconButton aria-label="Buka Folder" size="small"><FolderOpenIcon /></IconButton>
</Tooltip>
```

## Testing Strategy
Vitest + React Testing Library. Snapshot the TitleBar; assert a primary button
resolves to `#ef5350`; assert every interactive icon button has an
`aria-label` (accessibility proxy for "icons over text"). Visual check via `wails dev`.

## Boundaries
- **Always:** define any new color/radius/space in `tokens.ts` + `@theme` first;
  give every icon-only control an `aria-label`/`Tooltip`; keep the title bar's
  drag region (`--wails-draggable:drag`) working.
- **Ask first:** introducing a second accent color; changing the window radius;
  adding a component library alongside MUI.
- **Never:** hardcode hex values inside components; create a `tailwind.config.js`;
  break the frameless window (min/close must always work).

## Success Criteria
- [ ] Window is frameless with a working custom title bar (drag, minimize, close).
- [ ] Backgrounds, coral accent, and rounded corners match the reference image.
- [ ] Changing a token in `tokens.ts` updates both MUI and Tailwind output.
- [ ] Tailwind v4 utilities and MUI components coexist with no reset conflicts.
- [ ] `npm run typecheck` and `npm run lint` pass clean.

## Open Questions
- Reference shows a "Tauri + React" chip — we replace it with **"Wails + React"**.
- **Resolved:** TypeScript pinned to **5.x** (decided; not 7.x).
