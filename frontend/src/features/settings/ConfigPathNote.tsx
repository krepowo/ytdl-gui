import { useEffect, useState } from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import { tokens } from '../../theme/tokens'
import { api } from '../../api/client'
import type { AppInfo } from '../../api/types'

/**
 * Shows where the config file actually lives (the install folder, or the
 * %APPDATA% fallback when the install folder is read-only) plus the app and
 * bundled binary versions, so a user can confirm which install is running.
 */
export default function ConfigPathNote() {
  const [info, setInfo] = useState<AppInfo | null>(null)

  useEffect(() => {
    let active = true
    api
      .getAppInfo()
      .then((i) => {
        if (active) setInfo(i)
      })
      .catch(() => {
        /* non-critical */
      })
    return () => {
      active = false
    }
  }, [])

  if (!info) return null

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25 }}>
      <Typography variant="caption" sx={{ color: tokens.color.textMuted, wordBreak: 'break-all' }}>
        Config file: {info.configPath}
      </Typography>
      <Typography variant="caption" sx={{ color: tokens.color.textMuted }}>
        Version {info.version} • yt-dlp {info.ytDlpVersion} • ffmpeg {info.ffmpegVersion}
      </Typography>
    </Box>
  )
}
