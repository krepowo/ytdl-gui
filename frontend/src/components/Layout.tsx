import type { ReactNode } from 'react'
import Box from '@mui/material/Box'
import TitleBar from './TitleBar'
import { tokens } from '../theme/tokens'

/**
 * The app frame: a rounded, frameless window containing the custom title bar and
 * a scrollable content region. Every screen renders inside `children`.
 *
 * The OS window is transparent (see main.go), so this Box paints the visible
 * surface. A small inset keeps the rounded corners inside the window bounds.
 */
export default function Layout({ children }: { children: ReactNode }) {
  return (
    <Box
      sx={{
        height: '100vh',
        padding: '6px',
        boxSizing: 'border-box',
      }}
    >
      <Box
        sx={{
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          bgcolor: tokens.color.windowBg,
          borderRadius: `${tokens.radius.window}px`,
          overflow: 'hidden',
        }}
      >
        <TitleBar />

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
      </Box>
    </Box>
  )
}
