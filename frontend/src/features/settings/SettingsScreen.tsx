import { useCallback, useEffect, useRef, useState } from 'react'
import Alert from '@mui/material/Alert'
import Box from '@mui/material/Box'
import FormControl from '@mui/material/FormControl'
import IconButton from '@mui/material/IconButton'
import InputLabel from '@mui/material/InputLabel'
import MenuItem from '@mui/material/MenuItem'
import Select from '@mui/material/Select'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import Typography from '@mui/material/Typography'
import CloseIcon from '@mui/icons-material/Close'
import { tokens } from '../../theme/tokens'
import { api, errorMessage } from '../../api/client'
import type { Settings } from '../../api/types'
import DirPicker from './DirPicker'
import ConfigPathNote from './ConfigPathNote'

type Props = {
  onClose: () => void
}

/** The concurrency choices, fixed to the spec's 1–8 range. */
const CONCURRENCY_CHOICES = [1, 2, 3, 4, 5, 6, 7, 8]

/** Browsers yt-dlp can read cookies from; '' disables the option. */
const COOKIE_BROWSERS = ['', 'chrome', 'firefox', 'edge', 'brave', 'opera', 'vivaldi']

const DEBOUNCE_MS = 500

/**
 * Settings form bound to `GetSettings`/`SaveSettings`.
 *
 * Edits are saved automatically with a short debounce, so there is no explicit
 * Save button. Concurrency is a fixed 1–8 list, so an out-of-range value can
 * never reach `SaveSettings`.
 */
export default function SettingsScreen({ onClose }: Props) {
  const [form, setForm] = useState<Settings | null>(null)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  // Debounce timer + a flag so the initial load does not trigger a save.
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const loaded = useRef(false)

  useEffect(() => {
    let active = true
    api
      .getSettings()
      .then((s) => {
        if (!active) return
        setForm(s)
        loaded.current = true
      })
      .catch((err) => {
        if (active) setError(errorMessage(err))
      })
    return () => {
      active = false
      if (timer.current) clearTimeout(timer.current)
    }
  }, [])

  /** Applies a local change and schedules a debounced save. */
  const update = useCallback((patch: Partial<Settings>) => {
    setForm((prev) => {
      if (!prev) return prev
      const next = { ...prev, ...patch }
      if (loaded.current) {
        if (timer.current) clearTimeout(timer.current)
        timer.current = setTimeout(() => {
          api
            .saveSettings(next)
            .then(() => {
              setSaved(true)
              setError('')
            })
            .catch((err) => setError(errorMessage(err)))
        }, DEBOUNCE_MS)
      }
      return next
    })
  }, [])

  if (!form) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography variant="body2" sx={{ color: tokens.color.textMuted }}>
          Loading settings…
        </Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, p: 2 }}>
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        <Typography variant="subtitle1" sx={{ color: tokens.color.text, fontWeight: 600, flexGrow: 1 }}>
          Settings
        </Typography>
        <Tooltip title="Close settings">
          <IconButton aria-label="Close settings" onClick={onClose} size="small">
            <CloseIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>

      {error && <Alert severity="error">{error}</Alert>}
      {saved && !error && <Alert severity="success">Settings saved.</Alert>}

      <Box>
        <Typography variant="caption" sx={{ color: tokens.color.textMuted, display: 'block', mb: 0.5 }}>
          Download folder
        </Typography>
        <DirPicker value={form.downloadDir} onChange={(dir) => update({ downloadDir: dir })} />
      </Box>

      <Box sx={{ display: 'flex', gap: 2 }}>
        <FormControl size="small" sx={{ minWidth: 180 }}>
          <InputLabel id="concurrency-label">Concurrent downloads</InputLabel>
          <Select
            labelId="concurrency-label"
            label="Concurrent downloads"
            value={form.maxConcurrent}
            onChange={(e) => update({ maxConcurrent: Number(e.target.value) })}
            sx={{ borderRadius: `${tokens.radius.control}px` }}
          >
            {CONCURRENCY_CHOICES.map((n) => (
              <MenuItem key={n} value={n}>
                {n}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel id="mode-label">Default mode</InputLabel>
          <Select
            labelId="mode-label"
            label="Default mode"
            value={form.defaultMode}
            onChange={(e) => update({ defaultMode: e.target.value })}
            sx={{ borderRadius: `${tokens.radius.control}px` }}
          >
            <MenuItem value="video">Video</MenuItem>
            <MenuItem value="audio">Audio</MenuItem>
          </Select>
        </FormControl>
      </Box>

      <TextField
        size="small"
        label="Filename template"
        value={form.filenameTemplate}
        onChange={(e) => update({ filenameTemplate: e.target.value })}
        slotProps={{ htmlInput: { 'aria-label': 'Filename template' } }}
        sx={{
          '& .MuiOutlinedInput-root': { borderRadius: `${tokens.radius.control}px` },
        }}
      />

      <FormControl size="small" fullWidth>
        <InputLabel id="cookies-label">Cookies from browser</InputLabel>
        <Select
          labelId="cookies-label"
          label="Cookies from browser"
          value={form.cookiesBrowser}
          onChange={(e) => update({ cookiesBrowser: e.target.value })}
          sx={{ borderRadius: `${tokens.radius.control}px` }}
        >
          {COOKIE_BROWSERS.map((b) => (
            <MenuItem key={b || 'none'} value={b}>
              {b ? b.charAt(0).toUpperCase() + b.slice(1) : 'None'}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      <ConfigPathNote />
    </Box>
  )
}
