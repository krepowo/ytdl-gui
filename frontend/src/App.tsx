import Layout from './components/Layout'
import DownloadScreen from './features/download/DownloadScreen'

/**
 * App root. Task 11 mounts the download screen (URL input, media summary,
 * format picker); the queue list and settings screen follow in later tasks.
 */
export default function App() {
  return (
    <Layout>
      <DownloadScreen />
    </Layout>
  )
}
