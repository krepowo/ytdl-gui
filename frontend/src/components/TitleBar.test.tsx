import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, beforeEach } from 'vitest'
import TitleBar from './TitleBar'
import { renderWithTheme } from '../test/renderWithTheme'
import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'

describe('TitleBar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the app title', () => {
    renderWithTheme(<TitleBar />)
    expect(screen.getByText('Video Downloader')).toBeInTheDocument()
  })

  it('gives every icon-only control an accessible label', () => {
    renderWithTheme(<TitleBar />)
    for (const label of ['Minimalkan', 'Maksimalkan', 'Tutup']) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    }
  })

  it('minimises the window', async () => {
    renderWithTheme(<TitleBar />)
    await userEvent.click(screen.getByRole('button', { name: 'Minimalkan' }))
    expect(WindowMinimise).toHaveBeenCalledTimes(1)
  })

  it('toggles maximise', async () => {
    renderWithTheme(<TitleBar />)
    await userEvent.click(screen.getByRole('button', { name: 'Maksimalkan' }))
    expect(WindowToggleMaximise).toHaveBeenCalledTimes(1)
  })

  it('quits the app', async () => {
    renderWithTheme(<TitleBar />)
    await userEvent.click(screen.getByRole('button', { name: 'Tutup' }))
    expect(Quit).toHaveBeenCalledTimes(1)
  })
})
