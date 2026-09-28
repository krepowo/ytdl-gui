import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import TitleBar from './TitleBar'
import HeaderStrip from './HeaderStrip'
import { tokens } from '../theme/tokens'

/**
 * The app frame: a frameless window containing the custom title bar, a
 * scrollable content region, and an optional footer pinned below it.
 *
 * On Windows 11 the DWM draws the rounded corners and the drop shadow (see
 * main.go), so this box paints only the flat dark surface — no CSS radius and no
 * inset, which would otherwise leave a light fringe against DWM's corner.
 */
export default function Layout({
  children,
  footer,
}: {
  children: ReactNode
  footer?: ReactNode
}) {
  return (
    <Box
      sx={{
        height: '100vh',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: tokens.color.windowBg,
        overflow: 'hidden',
      }}
    >
      <TitleBar />

      <HeaderStrip />

      <Box
        component="main"
        sx={{
          flexGrow: 1,
          minHeight: 0,
          overflowY: 'auto',
          padding: `${tokens.space.page}px`,
          display: 'flex',
          flexDirection: 'column',
          gap: `${tokens.space.gap}px`,
        }}
      >
        {children}
      </Box>

      {footer}
    </Box>
  )
}
