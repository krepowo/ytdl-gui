import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { tokens } from './theme/tokens'

/**
 * Placeholder shell used to verify the toolchain (React 19 + MUI v9 + Tailwind v4).
 * Replaced by the real TitleBar/Layout in Task 3.
 */
export default function App() {
  return (
    <Box
      className="flex h-full items-center justify-center"
      sx={{ bgcolor: tokens.color.windowBg }}
    >
      <Typography variant="h5" sx={{ color: tokens.color.text }}>
        ytdl-gui
      </Typography>
    </Box>
  )
}
