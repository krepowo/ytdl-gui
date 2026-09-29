import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import Button from '@mui/material/Button'
import Stack from '@mui/material/Stack'
import AddToQueueIcon from '@mui/icons-material/AddToQueue'
import { tokens } from '../../theme/tokens'
import UrlInput from './UrlInput'
import MediaSummary from './MediaSummary'
import FormatPicker, { type FormatSelection } from './FormatPicker'
import { api, errorMessage } from '../../api/client'
import type { MediaInfo } from '../../api/types'

/**
 * The main screen: paste a URL from any supported site, probe it, choose a
 * format, and enqueue the download.
 *
 * The flow is deliberately two-step — probe first, then confirm — so the user
 * sees the detected source and the real available qualities before committing.
 */
export default function DownloadScreen() {
  const [url, setUrl] = useState('')
  const [probing, setProbing] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState<MediaInfo | null>(null)
  const [selection, setSelection] = useState<FormatSelection | null>(null)
  const [enqueuing, setEnqueuing] = useState(false)

  const reset = () => {
    setInfo(null)
    setSelection(null)
    setError('')
  }

  const handleProbe = async () => {
    const target = url.trim()
    if (!target) return

    setProbing(true)
    setError('')
    try {
      const result = await api.probeURL(target)
      setInfo(result)
    } catch (err) {
      setInfo(null)
      setSelection(null)
      setError(errorMessage(err))
    } finally {
      setProbing(false)
    }
  }

  const handleEnqueue = async () => {
    if (!info || !selection) return

    setEnqueuing(true)
    setError('')
    try {
      await api.startDownload({
        url: info.webpageUrl || url.trim(),
        mode: selection.mode,
        formatId: selection.formatId,
        needsMerge: selection.needsMerge,
        title: info.title,
      })
      // Clear the form so the user can paste the next link immediately.
      setUrl('')
      reset()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setEnqueuing(false)
    }
  }

  return (
    <Stack spacing={2}>
      <UrlInput
        value={url}
        onChange={(v) => {
          setUrl(v)
          if (info) reset()
        }}
        onDownload={handleProbe}
        probing={probing}
      />

      {error && (
        <Alert
          severity="error"
          variant="outlined"
          onClose={() => setError('')}
          sx={{ borderRadius: `${tokens.radius.control}px` }}
        >
          {error}
        </Alert>
      )}

      {info && (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          <MediaSummary info={info} />

          <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'center', flexWrap: 'wrap' }}>
            <FormatPicker info={info} onChange={setSelection} />
            <Box sx={{ flexGrow: 1 }} />
            <Button
              variant="contained"
              onClick={handleEnqueue}
              disabled={enqueuing || !selection}
              aria-label="Add to queue"
              startIcon={<AddToQueueIcon fontSize="small" />}
              sx={{
                borderRadius: `${tokens.radius.control}px`,
                textTransform: 'none',
                bgcolor: tokens.color.primary,
                '&:hover': { bgcolor: tokens.color.primaryHi },
              }}
            >
              Add to queue
            </Button>
          </Box>
        </Box>
      )}
    </Stack>
  )
}
