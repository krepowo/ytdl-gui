import { screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import App from './App'
import { renderWithTheme } from './test/renderWithTheme'
import { tokens } from './theme/tokens'

describe('App shell', () => {
  it('renders the window frame with the title bar', () => {
    renderWithTheme(<App />)
    expect(screen.getByText('Video Downloader')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tutup' })).toBeInTheDocument()
  })

  it('shows an empty-state message when there are no downloads', () => {
    renderWithTheme(<App />)
    expect(screen.getByText('Belum ada unduhan.')).toBeInTheDocument()
  })
})

describe('design tokens', () => {
  it('exposes the reference palette', () => {
    expect(tokens.color.primary).toBe('#ef5350')
    expect(tokens.color.windowBg).toBe('#121214')
    expect(tokens.color.success).toBe('#34d399')
  })
})
