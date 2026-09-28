import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import Layout from './components/Layout'
import { tokens } from './theme/tokens'

/**
 * App root. Task 3 establishes the window frame (Layout + TitleBar); the
 * download and settings screens are mounted here in later tasks.
 */
export default function App() {
  return (
    <Layout>
      <Box
        sx={{
          flexGrow: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Typography variant="body2" sx={{ color: tokens.color.textMuted }}>
          Belum ada unduhan.
        </Typography>
      </Box>
    </Layout>
  )
}
