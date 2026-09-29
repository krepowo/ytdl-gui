import Box from '@mui/material/Box'
import IconButton from '@mui/material/IconButton'
import TextField from '@mui/material/TextField'
import Tooltip from '@mui/material/Tooltip'
import CreateNewFolderIcon from '@mui/icons-material/CreateNewFolder'
import { tokens } from '../../theme/tokens'
import { api } from '../../api/client'

type Props = {
  value: string
  onChange: (dir: string) => void
}

/**
 * A read-only directory field with a folder icon that opens the native picker.
 *
 * The field is read-only (not disabled) so the path stays selectable and
 * screen-reader accessible while all edits go through the picker.
 */
export default function DirPicker({ value, onChange }: Props) {
  const handlePick = async () => {
    const dir = await api.pickDownloadDir()
    // An empty result means the user cancelled; leave the value untouched.
    if (dir) onChange(dir)
  }

  return (
    <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
      <TextField
        fullWidth
        size="small"
        value={value}
        slotProps={{ htmlInput: { readOnly: true, 'aria-label': 'Download folder' } }}
        sx={{
          '& .MuiOutlinedInput-root': {
            borderRadius: `${tokens.radius.control}px`,
            bgcolor: tokens.color.surface,
          },
        }}
      />
      <Tooltip title="Choose folder">
        <IconButton
          aria-label="Choose folder"
          onClick={handlePick}
          sx={{ color: tokens.color.textMuted }}
        >
          <CreateNewFolderIcon fontSize="small" />
        </IconButton>
      </Tooltip>
    </Box>
  )
}
