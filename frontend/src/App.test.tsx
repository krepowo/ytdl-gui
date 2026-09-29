import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import App from './App'
import { renderWithTheme } from './test/renderWithTheme'
import { installWailsGoMock } from './test/wailsGoMock'
import { tokens } from './theme/tokens'

// The queue hook subscribes to Wails events; stub the runtime module.
vi.mock('./wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(() => () => {}),
  EventsOff: vi.fn(),
  EventsEmit: vi.fn(),
}))

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

  it('mounts the queue list with its empty state', async () => {
    renderWithTheme(<App />)
    expect(screen.getByText(/antrean download/i)).toBeInTheDocument()
    expect(await screen.findByText('Belum ada unduhan.')).toBeInTheDocument()
  })

  it('opens the settings dialog from the storage bar', async () => {
    const user = userEvent.setup()
    renderWithTheme(<App />)
    await screen.findByRole('button', { name: 'Pengaturan' })

    await user.click(screen.getByRole('button', { name: 'Pengaturan' }))

    expect(await screen.findByText('Pengaturan')).toBeInTheDocument()
    expect(await screen.findByRole('textbox', { name: 'Folder unduhan' })).toBeInTheDocument()
  })
})

describe('design tokens', () => {
  it('exposes the reference palette', () => {
    expect(tokens.color.primary).toBe('#ef5350')
    expect(tokens.color.windowBg).toBe('#121214')
    expect(tokens.color.success).toBe('#34d399')
  })
})
