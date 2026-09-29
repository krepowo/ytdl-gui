import { useState } from 'react'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import CircularProgress from '@mui/material/CircularProgress'
import IconButton from '@mui/material/IconButton'
import InputAdornment from '@mui/material/InputAdornment'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import ContentPasteIcon from '@mui/icons-material/ContentPaste'
import DownloadIcon from '@mui/icons-material/Download'
import LinkIcon from '@mui/icons-material/Link'
import { tokens } from '../../theme/tokens'

type Props = {
  value: string
  onChange: (value: string) => void
  onDownload: () => void
  /** True while a probe is in flight: the action button is disabled and spins. */
  probing?: boolean
}

/**
 * The URL field and its Download action.
 *
 * The placeholder is deliberately source-agnostic: any URL yt-dlp supports is
 * accepted (1000+ sites plus the generic extractor), so it must not name a
 * single provider.
 */
export default function UrlInput({ value, onChange, onDownload, probing = false }: Props) {
  const [pasteError, setPasteError] = useState(false)
  const canDownload = value.trim().length > 0 && !probing

  const handleSubmit = () => {
    if (canDownload) onDownload()
  }

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText()
      if (text) onChange(text.trim())
      setPasteError(false)
    } catch {
      setPasteError(true)
    }
  }

  return (
    <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'flex-start' }}>
      <TextField
        fullWidth
        size="small"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter') handleSubmit()
        }}
        placeholder="Paste a video link from any site…"
        sx={{
          '& .MuiOutlinedInput-root': {
            borderRadius: `${tokens.radius.control}px`,
            bgcolor: tokens.color.surface,
          },
        }}
        slotProps={{
          htmlInput: { 'aria-label': 'Video URL' },
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <LinkIcon fontSize="small" sx={{ color: tokens.color.textMuted }} />
              </InputAdornment>
            ),
            endAdornment: (
              <InputAdornment position="end">
                <Tooltip title={pasteError ? 'Paste failed — allow clipboard access' : 'Paste from clipboard'}>
                  <IconButton
                    size="small"
                    aria-label="Paste from clipboard"
                    onClick={handlePaste}
                  >
                    <ContentPasteIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </InputAdornment>
            ),
          },
        }}
      />

      <Button
        variant="contained"
        onClick={handleSubmit}
        disabled={!canDownload}
        aria-label="Download"
        startIcon={
          probing ? (
            <CircularProgress size={16} color="inherit" />
          ) : (
            <DownloadIcon fontSize="small" />
          )
        }
        sx={{
          flexShrink: 0,
          height: 40,
          px: 2.5,
          borderRadius: `${tokens.radius.control}px`,
          textTransform: 'none',
          bgcolor: tokens.color.primary,
          '&:hover': { bgcolor: tokens.color.primaryHi },
        }}
      >
        Download
      </Button>
    </Box>
  )
}
