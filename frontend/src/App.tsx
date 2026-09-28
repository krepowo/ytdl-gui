import { useState } from 'react'
import Alert from '@mui/material/Alert'
import Dialog from '@mui/material/Dialog'
import DialogContent from '@mui/material/DialogContent'
import Snackbar from '@mui/material/Snackbar'
import Stack from '@mui/material/Stack'
import { tokens } from './theme/tokens'
import Layout from './components/Layout'
import DownloadScreen from './features/download/DownloadScreen'
import QueueList from './features/download/QueueList'
import StorageBar from './features/download/StorageBar'
import SettingsScreen from './features/settings/SettingsScreen'
import { useDownloadQueue } from './features/download/useDownloadQueue'

/**
 * App root. The download screen sits on top of the live queue list; both share
 * one `useDownloadQueue` instance so a job enqueued by the form appears in the
 * list immediately via the `job:added` event. The storage bar is pinned as the
 * window footer, and settings open in a modal dialog.
 */
export default function App() {
  const queue = useDownloadQueue()
  const [settingsOpen, setSettingsOpen] = useState(false)

  return (
    <>
      <Layout footer={<StorageBar onOpenSettings={() => setSettingsOpen(true)} />}>
        <Stack spacing={3}>
          <DownloadScreen />
          <QueueList
            jobs={queue.jobs}
            onPause={queue.pause}
            onResume={queue.resume}
            onCancel={queue.cancel}
            onOpen={queue.open}
            onRetry={queue.retry}
            onRemove={queue.remove}
          />
        </Stack>
      </Layout>

      <Dialog
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        maxWidth="sm"
        fullWidth
        slotProps={{
          paper: {
            sx: {
              bgcolor: tokens.color.windowBg,
              backgroundImage: 'none',
              borderRadius: `${tokens.radius.window}px`,
              border: `1px solid ${tokens.color.border}`,
            },
          },
        }}
      >
        <DialogContent>
          <SettingsScreen onClose={() => setSettingsOpen(false)} />
        </DialogContent>
      </Dialog>

      <Snackbar
        open={Boolean(queue.lastError)}
        autoHideDuration={6000}
        onClose={queue.clearError}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="error" variant="filled" onClose={queue.clearError}>
          {queue.lastError}
        </Alert>
      </Snackbar>
    </>
  )
}
