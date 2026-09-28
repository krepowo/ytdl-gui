/**
 * Design tokens — the single source of truth for the app's visual language.
 *
 * The same values are mirrored in `src/index.css` inside Tailwind v4's `@theme`
 * block (Tailwind v4 is CSS-first, so it cannot import this file). When you change
 * a token here, update `index.css` too — they must stay in sync.
 *
 * Reference design: dark charcoal surfaces with a coral accent.
 */
export const tokens = {
  color: {
    appBg: '#0a0a0c',
    windowBg: '#121214',
    surface: '#18181b',
    surfaceAlt: '#1f1f24',
    titleBar: '#232329',
    border: '#27272a',
    primary: '#ef5350',
    primaryHi: '#f87171',
    success: '#34d399',
    text: '#f3f4f6',
    textMuted: '#9ca3af',
  },
  radius: {
    window: 18,
    control: 12,
    card: 14,
  },
  space: {
    page: 24,
    gap: 16,
  },
} as const

export type Tokens = typeof tokens
