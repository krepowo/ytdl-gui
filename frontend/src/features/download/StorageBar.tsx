import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import FolderOpenIcon from '@mui/icons-material/FolderOpen'
import SettingsIcon from '@mui/icons-material/Settings'
import { tokens } from '../../theme/tokens'
import { api } from '../../api/client'
import { formatBytes } from '../../utils/format'

type Props = {
  onOpenSettings: () => void
}

/**
 * Footer bar showing where downloads go, the free space on that volume, and
 * icon actions to open the folder or the settings screen.
 *
 * Free space is read once per path; it is not polled (downloads change it, but a
 * stale figure is harmless and the value refreshes when the path changes).
 */
export default function StorageBar({ onOpenSettings }: Props) {
  const [dir, setDir] = useState('')
  const [free, setFree] = useState<number | null>(null)

  useEffect(() => {
    let active = true
    api
      .getSettings()
      .then((s) => {
        if (active) setDir(s.downloadDir)
      })
      .catch(() => {
        /* the bar is non-critical; stay quiet on failure */
      })
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (!dir) return
    let active = true
    api
      .getDiskFree(dir)
      .then((bytes) => {
        if (active) setFree(bytes)
      })
      .catch(() => {
        if (active) setFree(null)
      })
    return () => {
      active = false
    }
  }, [dir])

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 2,
        py: 1,
        borderTop: `1px solid ${tokens.color.border}`,
        bgcolor: tokens.color.surface,
      }}
    >
      <Typography
        variant="caption"
        sx={{
          color: tokens.color.textMuted,
          flexGrow: 1,
          minWidth: 0,
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {dir}
        {free !== null ? ` • ${formatBytes(free)} available` : ''}
      </Typography>

      <Tooltip title="Open folder">
        <IconButton size="small" aria-label="Open folder" onClick={() => api.openDownloadDir()}>
          <FolderOpenIcon fontSize="small" />
        </IconButton>
      </Tooltip>

      <Tooltip title="Settings">
        <IconButton size="small" aria-label="Settings" onClick={onOpenSettings}>
          <SettingsIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Box>
  )
}
