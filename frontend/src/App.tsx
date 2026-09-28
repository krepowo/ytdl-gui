import Snackbar from '@mui/material/Snackbar'
import Alert from '@mui/material/Alert'
import Stack from '@mui/material/Stack'
import Layout from './components/Layout'
import DownloadScreen from './features/download/DownloadScreen'
import QueueList from './features/download/QueueList'
import { useDownloadQueue } from './features/download/useDownloadQueue'

/**
 * App root. The download screen sits on top of the live queue list; both share
 * one `useDownloadQueue` instance so a job enqueued by the form appears in the
 * list immediately via the `job:added` event.
 */
export default function App() {
  const queue = useDownloadQueue()

  return (
    <Layout>
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
    </Layout>
  )
}
