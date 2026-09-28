import { render, screen } from '@testing-library/react'
import { ThemeProvider } from '@mui/material/styles'
import { describe, it, expect } from 'vitest'
import App from './App'
import { muiTheme } from './theme/muiTheme'
import { tokens } from './theme/tokens'

function renderApp() {
  return render(
    <ThemeProvider theme={muiTheme}>
      <App />
    </ThemeProvider>,
  )
}

describe('App shell', () => {
  it('renders the app name', () => {
    renderApp()
    expect(screen.getByText('ytdl-gui')).toBeInTheDocument()
  })
})

describe('design tokens', () => {
  it('exposes the reference palette', () => {
    expect(tokens.color.primary).toBe('#ef5350')
    expect(tokens.color.windowBg).toBe('#121214')
    expect(tokens.color.success).toBe('#34d399')
  })
})
