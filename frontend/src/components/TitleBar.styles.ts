import { styled } from '@mui/material/styles'
import type { BoxProps } from '@mui/material/Box'
import Box from '@mui/material/Box'
import { tokens } from '../theme/tokens'

/**
 * The custom title bar for the frameless window.
 *
 * The bar itself is the drag region (`--wails-draggable: drag`); the window
 * control buttons opt out with `--wails-draggable: no-drag` so clicks land on
 * them instead of starting a window drag.
 */
export const TitleBarRoot = styled(Box)<BoxProps>(({ theme }) => ({
  height: 40,
  flexShrink: 0,
  display: 'flex',
  alignItems: 'center',
  gap: theme.spacing(1),
  paddingLeft: theme.spacing(1.5),
  paddingRight: theme.spacing(1),
  backgroundColor: tokens.color.titleBar,
  borderTopLeftRadius: tokens.radius.window,
  borderTopRightRadius: tokens.radius.window,
  userSelect: 'none',
  '--wails-draggable': 'drag',
}))

/** A circular window-control button that sits in the title bar. */
export const WindowControl = styled('button')(({ theme }) => ({
  width: 28,
  height: 28,
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  border: 'none',
  borderRadius: theme.shape.borderRadius,
  background: 'transparent',
  color: tokens.color.textMuted,
  cursor: 'pointer',
  transition: 'background-color 120ms ease, color 120ms ease',
  '--wails-draggable': 'no-drag',
  '&:hover': {
    backgroundColor: 'rgba(255,255,255,0.06)',
    color: tokens.color.text,
  },
  '&:focus-visible': {
    outline: `2px solid ${tokens.color.primary}`,
    outlineOffset: 2,
  },
}))
