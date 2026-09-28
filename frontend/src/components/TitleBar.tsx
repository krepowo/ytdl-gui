import Avatar from '@mui/material/Avatar'
import Box from '@mui/material/Box'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import CloseIcon from '@mui/icons-material/Close'
import CropSquareIcon from '@mui/icons-material/CropSquare'
import RemoveIcon from '@mui/icons-material/Remove'
import FilterVintageIcon from '@mui/icons-material/FilterVintage'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'
import { tokens } from '../theme/tokens'
import { TitleBarRoot, WindowControl } from './TitleBar.styles'

/**
 * Custom dark title bar for the frameless window: app mark, title, and window
 * controls (minimise / maximise / close). Dragging is handled by CSS via the
 * `--wails-draggable` property set in TitleBar.styles.ts.
 */
export default function TitleBar() {
  return (
    <TitleBarRoot>
      <Avatar
        sx={{
          width: 22,
          height: 22,
          bgcolor: tokens.color.primary,
          color: '#fff',
        }}
      >
        <FilterVintageIcon sx={{ fontSize: 14 }} />
      </Avatar>

      <Typography
        variant="subtitle2"
        sx={{ fontWeight: 600, color: tokens.color.text, letterSpacing: 0.2 }}
      >
        Video Downloader
      </Typography>

      <Box sx={{ flexGrow: 1 }} />

      <Tooltip title="Minimalkan">
        <WindowControl aria-label="Minimalkan" onClick={() => WindowMinimise()}>
          <RemoveIcon sx={{ fontSize: 16 }} />
        </WindowControl>
      </Tooltip>

      <Tooltip title="Maksimalkan">
        <WindowControl aria-label="Maksimalkan" onClick={() => WindowToggleMaximise()}>
          <CropSquareIcon sx={{ fontSize: 14 }} />
        </WindowControl>
      </Tooltip>

      <Tooltip title="Tutup">
        <WindowControl
          aria-label="Tutup"
          onClick={() => Quit()}
          sx={{
            '&:hover': {
              backgroundColor: tokens.color.primary,
              color: '#fff',
            },
          }}
        >
          <CloseIcon sx={{ fontSize: 16 }} />
        </WindowControl>
      </Tooltip>
    </TitleBarRoot>
  )
}
