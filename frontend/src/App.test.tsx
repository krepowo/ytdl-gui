import { screen } from '@testing-library/react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import App from './App'
import { renderWithTheme } from './test/renderWithTheme'
import { installWailsGoMock } from './test/wailsGoMock'
import { tokens } from './theme/tokens'

describe('App shell', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    installWailsGoMock()
  })

  it('renders the window frame with the title bar', () => {
    renderWithTheme(<App />)
    expect(screen.getByText('Video Downloader')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tutup' })).toBeInTheDocument()
  })

  it('mounts the download screen with its URL input', () => {
    renderWithTheme(<App />)
    expect(screen.getByRole('textbox', { name: 'Tautan video' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /unduh/i })).toBeInTheDocument()
  })
})

describe('design tokens', () => {
  it('exposes the reference palette', () => {
    expect(tokens.color.primary).toBe('#ef5350')
    expect(tokens.color.windowBg).toBe('#121214')
    expect(tokens.color.success).toBe('#34d399')
  })
})
