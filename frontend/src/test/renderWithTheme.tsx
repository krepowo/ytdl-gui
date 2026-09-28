import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { muiTheme } from '../theme/muiTheme'

/** Renders a component inside the app's MUI theme (mirrors main.tsx). */
export function renderWithTheme(ui: ReactElement) {
  return render(<ThemeProvider theme={muiTheme}>{ui}</ThemeProvider>)
}
