import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { tokens } from '../../theme/tokens'
import type { Job } from '../../api/types'
import QueueItem from './QueueItem'

type Props = {
  jobs: Job[]
  onPause: (id: string) => void
  onResume: (id: string) => void
  onCancel: (id: string) => void
  onOpen: (path: string) => void
  onRetry: (id: string) => void
  onRemove: (id: string) => void
}

/**
 * The "Antrean Download" section: a header with live counts and the list of
 * download items. The counts come from the passed-in jobs, so they update the
 * moment an event changes a job's state.
 */
export default function QueueList({
  jobs,
  onPause,
  onResume,
  onCancel,
  onOpen,
  onRetry,
  onRemove,
}: Props) {
  const queued = jobs.filter((j) => j.state === 'queued').length
  const downloading = jobs.filter((j) => j.state === 'downloading').length

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1.5 }}>
        <Typography variant="subtitle2" sx={{ color: tokens.color.text, fontWeight: 600 }}>
          Antrean Download
        </Typography>
        {jobs.length > 0 && (
          <Typography variant="caption" sx={{ color: tokens.color.textMuted }}>
            {queued} antrean • {downloading} sedang diunduh
          </Typography>
        )}
      </Box>

      {jobs.length === 0 ? (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            py: 5,
          }}
        >
          <Typography variant="body2" sx={{ color: tokens.color.textMuted }}>
            Belum ada unduhan.
          </Typography>
        </Box>
      ) : (
        <Stack spacing={1}>
          {jobs.map((job) => (
            <QueueItem
              key={job.id}
              job={job}
              onPause={onPause}
              onResume={onResume}
              onCancel={onCancel}
              onOpen={onOpen}
              onRetry={onRetry}
              onRemove={onRemove}
            />
          ))}
        </Stack>
      )}
    </Box>
  )
}
