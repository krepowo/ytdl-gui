import Box from '@mui/material/Box'
import Chip from '@mui/material/Chip'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { tokens } from '../../theme/tokens'
import type { MediaInfo } from '../../api/types'

type Props = {
  info: MediaInfo
}

/**
 * Post-probe summary of the media: thumbnail, title, source badge, uploader and
 * duration.
 *
 * Every field except the title is optional (generic pages and live streams often
 * lack a thumbnail or duration), so each is rendered only when present — the UI
 * must never show "undefined" or a broken image.
 */
export default function MediaSummary({ info }: Props) {
  const hasThumbnail = info.thumbnail.trim().length > 0
  const hasUploader = info.uploader.trim().length > 0
  const hasDuration = info.duration > 0
  const source = info.extractor.trim()

  return (
    <Box
      sx={{
        display: 'flex',
        gap: 2,
        p: 2,
        bgcolor: tokens.color.surface,
        border: `1px solid ${tokens.color.border}`,
        borderRadius: `${tokens.radius.card}px`,
      }}
    >
      {hasThumbnail && (
        <Box
          component="img"
          src={info.thumbnail}
          alt={info.title}
          sx={{
            width: 128,
            height: 72,
            objectFit: 'cover',
            borderRadius: `${tokens.radius.control}px`,
            flexShrink: 0,
            bgcolor: tokens.color.surfaceAlt,
          }}
        />
      )}

      <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.75 }}>
        <Typography
          variant="subtitle2"
          sx={{
            color: tokens.color.text,
            fontWeight: 600,
            display: '-webkit-box',
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {info.title}
        </Typography>

        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
          {source && (
            <Chip
              label={source}
              size="small"
              sx={{
                height: 20,
                textTransform: 'capitalize',
                bgcolor: tokens.color.surfaceAlt,
                color: tokens.color.textMuted,
                fontSize: 11,
              }}
            />
          )}

          {info.isLive && (
            <Chip
              label="LIVE"
              size="small"
              sx={{
                height: 20,
                bgcolor: tokens.color.primary,
                color: '#fff',
                fontWeight: 700,
                fontSize: 11,
              }}
            />
          )}

          {hasDuration && (
            <Typography variant="caption" sx={{ color: tokens.color.textMuted }}>
              {formatDuration(info.duration)}
            </Typography>
          )}

          {hasUploader && (
            <Typography
              variant="caption"
              sx={{ color: tokens.color.textMuted, overflow: 'hidden', textOverflow: 'ellipsis' }}
            >
              {info.uploader}
            </Typography>
          )}
        </Stack>
      </Box>
    </Box>
  )
}

/**
 * Formats a duration in seconds as mm:ss, or h:mm:ss past an hour. Zero (an
 * unknown duration) returns an empty string so callers can hide it.
 */
export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return ''
  const total = Math.round(seconds)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`
  return `${pad(m)}:${pad(s)}`
}
