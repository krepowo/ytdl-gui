import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import { tokens } from '../theme/tokens'

/**
 * A slim strip directly under the title bar, mirroring the reference design's
 * header row. The reference shows a "Tauri + React" chip here; per
 * docs/SPEC-ui-shell.md (Open Questions) this project ships the same chip
 * reading "Wails + React".
 *
 * The reference's left-hand hotkey hint is intentionally omitted: no spec defines
 * a global-hotkey overlay window, so there is nothing to advertise yet.
 */
export default function HeaderStrip() {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'flex-end',
        px: `${tokens.space.page}px`,
        pt: 1.5,
      }}
    >
      <Chip
        label="Wails + React"
        size="small"
        sx={{
          bgcolor: tokens.color.surfaceAlt,
          color: tokens.color.textMuted,
          borderRadius: `${tokens.radius.control}px`,
          fontSize: 11,
          height: 22,
        }}
      />
    </Box>
  )
}
