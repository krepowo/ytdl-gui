import { createTheme, type Theme } from '@mui/material/styles'
import { tokens } from './tokens'

/**
 * MUI v9 theme built from the shared design tokens.
 *
 * We enable CSS-variables mode so MUI emits `--mui-*` custom properties derived
 * with `color-mix()`. This lets MUI components and Tailwind v4 utilities resolve
 * the same values, avoiding two competing colour systems.
 */
export const muiTheme: Theme = createTheme({
  cssVariables: true,
  palette: {
    mode: 'dark',
    primary: {
      main: tokens.color.primary,
      light: tokens.color.primaryHi,
    },
    success: { main: tokens.color.success },
    background: {
      default: tokens.color.windowBg,
      paper: tokens.color.surface,
    },
    text: {
      primary: tokens.color.text,
      secondary: tokens.color.textMuted,
    },
    divider: tokens.color.border,
  },
  shape: {
    borderRadius: tokens.radius.control,
  },
  typography: {
    fontFamily:
      '"Inter", "Segoe UI", system-ui, -apple-system, sans-serif',
  },
})
