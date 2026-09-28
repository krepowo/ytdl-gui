import { useEffect, useMemo, useState } from 'react'
import Box from '@mui/material/Box'
import MenuItem from '@mui/material/MenuItem'
import Select from '@mui/material/Select'
import ToggleButton from '@mui/material/ToggleButton'
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup'
import Typography from '@mui/material/Typography'
import AudiotrackIcon from '@mui/icons-material/Audiotrack'
import VideocamIcon from '@mui/icons-material/Videocam'
import { tokens } from '../../theme/tokens'
import type { FormatOption, MediaInfo } from '../../api/types'
import { MODE_AUDIO, MODE_VIDEO } from '../../api/types'

/** The chosen format, reported to the parent so it can start the download. */
export type FormatSelection = {
  mode: string
  formatId: string
  needsMerge: boolean
}

type Props = {
  info: MediaInfo
  onChange: (selection: FormatSelection) => void
}

/**
 * Video/audio toggle plus a quality dropdown built from the formats the site
 * actually returned. There is no fixed quality ladder: a site offering only a
 * combined 360p stream shows one option, while one offering DASH streams shows
 * every resolution (and the engine merges audio when needed).
 */
/** Renders "label • ext", but drops the ext when the label already mentions it
 * (which produced "mp4 • mp4" and "mp4 Default, low • mp4"). */
function optionText(o: FormatOption): string {
  if (!o.ext) return o.label
  const label = o.label.toLowerCase()
  const ext = o.ext.toLowerCase()
  // Split on non-letters so "mp4" in "mp4 Default, low" still matches.
  if (label === ext || label.split(/[^a-z0-9]+/).includes(ext)) return o.label
  return `${o.label} • ${o.ext}`
}

export default function FormatPicker({ info, onChange }: Props) {
  const hasVideo = info.videoOptions.length > 0
  const hasAudio = info.audioOptions.length > 0

  // A live stream has no selectable qualities; default to the best video.
  const initialMode = info.isLive ? MODE_VIDEO : hasVideo ? MODE_VIDEO : MODE_AUDIO
  const [mode, setMode] = useState<string>(initialMode)
  const [formatId, setFormatId] = useState<string>('')

  const options: FormatOption[] = useMemo(() => {
    if (info.isLive) return []
    return mode === MODE_AUDIO ? info.audioOptions : info.videoOptions
  }, [mode, info.audioOptions, info.videoOptions, info.isLive])

  // The selection is DERIVED, not synced: the stored formatId is honoured when
  // it still exists in the current option list, otherwise the first option wins.
  // Deriving avoids a setState-in-effect cascade.
  const selected: FormatOption | undefined = useMemo(
    () => options.find((o) => o.formatId === formatId) ?? options[0],
    [options, formatId],
  )

  // Report the derived selection upward so the parent always has a valid choice.
  useEffect(() => {
    if (!selected) return
    onChange({
      mode,
      formatId: selected.formatId,
      needsMerge: mode === MODE_VIDEO ? selected.needsMerge : false,
    })
    // onChange is intentionally excluded: it is a stable callback from the parent.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode, selected])

  const handleModeChange = (_: unknown, next: string | null) => {
    if (!next) return
    setMode(next)
    // Clear the format so the derived selection falls back to the new mode's
    // first option.
    setFormatId('')
  }

  if (info.isLive) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography variant="caption" sx={{ color: tokens.color.textMuted }}>
          Siaran langsung — kualitas dipilih otomatis (LIVE)
        </Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'center', flexWrap: 'wrap' }}>
      <ToggleButtonGroup
        size="small"
        exclusive
        value={mode}
        onChange={handleModeChange}
        aria-label="Mode unduhan"
      >
        <ToggleButton value={MODE_VIDEO} disabled={!hasVideo} aria-label="Video">
          <VideocamIcon fontSize="small" sx={{ mr: 0.5 }} />
          Video
        </ToggleButton>
        <ToggleButton value={MODE_AUDIO} disabled={!hasAudio} aria-label="Audio">
          <AudiotrackIcon fontSize="small" sx={{ mr: 0.5 }} />
          Audio
        </ToggleButton>
      </ToggleButtonGroup>

      <Select
        size="small"
        value={selected?.formatId ?? ''}
        onChange={(e) => setFormatId(e.target.value)}
        aria-label="Kualitas"
        renderValue={(v) => {
          const o = options.find((x) => x.formatId === v)
          return o ? optionText(o) : ''
        }}
        sx={{
          minWidth: 180,
          borderRadius: `${tokens.radius.control}px`,
          bgcolor: tokens.color.surface,
        }}
      >
        {options.map((o) => (
          <MenuItem key={o.formatId} value={o.formatId}>
            {optionText(o)}
          </MenuItem>
        ))}
      </Select>
    </Box>
  )
}
