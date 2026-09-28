import { type ReactNode } from 'react'
import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import LinearProgress from '@mui/material/LinearProgress'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import DeleteIcon from '@mui/icons-material/Delete'
import DownloadIcon from '@mui/icons-material/Download'
import ErrorIcon from '@mui/icons-material/Error'
import FolderOpenIcon from '@mui/icons-material/FolderOpen'
import PauseIcon from '@mui/icons-material/Pause'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import RefreshIcon from '@mui/icons-material/Refresh'
import ScheduleIcon from '@mui/icons-material/Schedule'
import CloseIcon from '@mui/icons-material/Close'
import { tokens } from '../../theme/tokens'
import type { Job, JobState } from '../../api/types'
import { clampPercent, formatBytes, formatEta, formatSpeed } from '../../utils/format'

type Props = {
  job: Job
  onPause: (id: string) => void
  onResume: (id: string) => void
  onCancel: (id: string) => void
  onOpen: (path: string) => void
  onRetry: (id: string) => void
  onRemove: (id: string) => void
}

/** Per-state presentation: status icon, colour, and whether a bar is shown. */
const statusMeta: Record<JobState, { icon: ReactNode; color: string; showBar: boolean }> = {
  downloading: {
    icon: <DownloadIcon fontSize="small" />,
    color: tokens.color.primary,
    showBar: true,
  },
  queued: {
    icon: <ScheduleIcon fontSize="small" />,
    color: tokens.color.textMuted,
    showBar: false,
  },
  paused: {
    icon: <PauseIcon fontSize="small" />,
    color: tokens.color.textMuted,
    showBar: true,
  },
  completed: {
    icon: <CheckCircleIcon fontSize="small" />,
    color: tokens.color.success,
    showBar: false,
  },
  error: {
    icon: <ErrorIcon fontSize="small" />,
    color: tokens.color.primary,
    showBar: false,
  },
  canceled: {
    icon: <CloseIcon fontSize="small" />,
    color: tokens.color.textMuted,
    showBar: false,
  },
}

/** The human status line, e.g. "Mengunduh • 42.0 MB / 100.0 MB • 42.3%". */
export function statusLine(job: Job): string {
  switch (job.state) {
    case 'queued':
      return 'Menunggu'
    case 'paused':
      return `Dijeda • ${clampPercent(job.percent).toFixed(1)}%`
    case 'completed':
      return 'Selesai'
    case 'canceled':
      return 'Dibatalkan'
    case 'error':
      return job.error || 'Gagal'
    case 'downloading': {
      const parts = ['Mengunduh']
      if (job.totalBytes > 0) {
        parts.push(`${formatBytes(job.downloadedBytes)} / ${formatBytes(job.totalBytes)}`)
      } else if (job.downloadedBytes > 0) {
        parts.push(formatBytes(job.downloadedBytes))
      }
      parts.push(`${clampPercent(job.percent).toFixed(1)}%`)
      const speed = formatSpeed(job.speedBps)
      if (speed) parts.push(speed)
      const eta = formatEta(job.etaSec)
      if (eta) parts.push(`ETA ${eta}`)
      return parts.join(' • ')
    }
    default:
      return ''
  }
}

/**
 * One row in the download queue: a status icon, the title, a status line, an
 * optional progress bar, and the icon actions valid for the current state.
 *
 * Actions are icon-only (with tooltips and aria-labels) to match the reference
 * design; the set changes with the state so only valid operations are offered.
 */
export default function QueueItem({
  job,
  onPause,
  onResume,
  onCancel,
  onOpen,
  onRetry,
  onRemove,
}: Props) {
  const meta = statusMeta[job.state]
  const title = job.title.trim() || job.url

  return (
    <Box
      sx={{
        display: 'flex',
        gap: 1.5,
        alignItems: 'flex-start',
        p: 1.5,
        bgcolor: tokens.color.surface,
        border: `1px solid ${tokens.color.border}`,
        borderRadius: `${tokens.radius.card}px`,
      }}
    >
      <Box sx={{ color: meta.color, display: 'flex', pt: 0.25 }}>{meta.icon}</Box>

      <Box sx={{ flexGrow: 1, minWidth: 0 }}>
        <Typography
          variant="body2"
          sx={{
            color: tokens.color.text,
            fontWeight: 500,
            whiteSpace: 'nowrap',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {title}
        </Typography>

        <Typography variant="caption" sx={{ color: tokens.color.textMuted }}>
          {statusLine(job)}
        </Typography>

        {meta.showBar && (
          <LinearProgress
            variant="determinate"
            value={clampPercent(job.percent)}
            aria-label="Progres unduhan"
            sx={{
              mt: 1,
              height: 6,
              borderRadius: 3,
              bgcolor: tokens.color.surfaceAlt,
              '& .MuiLinearProgress-bar': { bgcolor: tokens.color.primary },
            }}
          />
        )}
      </Box>

      <Box sx={{ display: 'flex', gap: 0.5, flexShrink: 0 }}>
        {job.state === 'downloading' && (
          <Tooltip title="Jeda">
            <IconButton size="small" aria-label="Jeda" onClick={() => onPause(job.id)}>
              <PauseIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {job.state === 'paused' && (
          <Tooltip title="Lanjutkan">
            <IconButton size="small" aria-label="Lanjutkan" onClick={() => onResume(job.id)}>
              <PlayArrowIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {(job.state === 'downloading' || job.state === 'paused' || job.state === 'queued') && (
          <Tooltip title="Batalkan">
            <IconButton size="small" aria-label="Batalkan" onClick={() => onCancel(job.id)}>
              <CloseIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {job.state === 'completed' && (
          <Tooltip title="Buka">
            <IconButton size="small" aria-label="Buka" onClick={() => onOpen(job.outputPath)}>
              <FolderOpenIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {job.state === 'error' && (
          <Tooltip title="Coba lagi">
            <IconButton size="small" aria-label="Coba lagi" onClick={() => onRetry(job.id)}>
              <RefreshIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}

        {(job.state === 'completed' || job.state === 'error' || job.state === 'canceled') && (
          <Tooltip title="Hapus">
            <IconButton size="small" aria-label="Hapus" onClick={() => onRemove(job.id)}>
              <DeleteIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        )}
      </Box>
    </Box>
  )
}
