import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SettingsScreen from './SettingsScreen'
import { renderWithTheme } from '../../test/renderWithTheme'
import { installWailsGoMock } from '../../test/wailsGoMock'
import type { Settings } from '../../api/types'

function settings(overrides: Partial<Settings> = {}): Settings {
  return {
    downloadDir: 'D:\\Downloads',
    maxConcurrent: 2,
    defaultMode: 'video',
    defaultQuality: 'best',
    filenameTemplate: '%(title)s.%(ext)s',
    cookiesBrowser: '',
    ...overrides,
  }
}

describe('SettingsScreen', () => {
  let mocks: ReturnType<typeof installWailsGoMock>

  beforeEach(() => {
    vi.clearAllMocks()
    mocks = installWailsGoMock()
    mocks.GetSettings.mockResolvedValue(settings())
    mocks.GetAppInfo.mockResolvedValue({
      version: '0.1.0',
      configPath: 'C:\\app\\config.json',
      ytDlpVersion: '2026.08.19',
      ffmpegVersion: '9.0',
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads settings into the form', async () => {
    renderWithTheme(<SettingsScreen onClose={vi.fn()} />)

    expect(await screen.findByRole('textbox', { name: 'Folder unduhan' })).toHaveValue('D:\\Downloads')
  })

  it('shows the max-concurrency control with the loaded value', async () => {
    renderWithTheme(<SettingsScreen onClose={vi.fn()} />)

    const concurrency = await screen.findByRole('combobox', { name: /unduhan bersamaan/i })
    expect(concurrency).toHaveTextContent('2')
  })

  it('persists a concurrency change via SaveSettings (debounced)', async () => {
    renderWithTheme(<SettingsScreen onClose={vi.fn()} />)

    const concurrency = await screen.findByRole('combobox', { name: /unduhan bersamaan/i })
    expect(concurrency).toHaveTextContent('2')

    fireEvent.mouseDown(concurrency)
    const option = await screen.findByRole('option', { name: '4' })
    fireEvent.click(option)

    // Nothing saved until the debounce elapses.
    expect(mocks.SaveSettings).not.toHaveBeenCalled()

    await waitFor(() => expect(mocks.SaveSettings).toHaveBeenCalled(), { timeout: 2000 })
    const saved = mocks.SaveSettings.mock.calls.at(-1)?.[0] as Settings
    expect(saved.maxConcurrent).toBe(4)
  })

  it('does not save a concurrency outside 1-8', async () => {
    // The control is a fixed 1-8 list, so an out-of-range value cannot be
    // selected. Assert the option set to prove the boundary.
    renderWithTheme(<SettingsScreen onClose={vi.fn()} />)
    const concurrency = await screen.findByRole('combobox', { name: /unduhan bersamaan/i })

    fireEvent.mouseDown(concurrency)
    expect(await screen.findByRole('option', { name: '1' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: '8' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: '9' })).not.toBeInTheDocument()
    expect(screen.queryByRole('option', { name: '0' })).not.toBeInTheDocument()
  })

  it('updates the download dir from the folder picker', async () => {
    mocks.PickDownloadDir.mockResolvedValue('E:\\Media')
    const user = userEvent.setup()
    renderWithTheme(<SettingsScreen onClose={vi.fn()} />)
    await screen.findByRole('textbox', { name: 'Folder unduhan' })

    await user.click(screen.getByRole('button', { name: /pilih folder/i }))

    await waitFor(() =>
      expect(screen.getByRole('textbox', { name: 'Folder unduhan' })).toHaveValue('E:\\Media'),
    )
  })

  it('shows the active config path', async () => {
    renderWithTheme(<SettingsScreen onClose={vi.fn()} />)
    expect(await screen.findByText(/C:\\app\\config\.json/)).toBeInTheDocument()
  })

  it('closes via the close button', async () => {
    const onClose = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<SettingsScreen onClose={onClose} />)
    await screen.findByRole('textbox', { name: 'Folder unduhan' })

    await user.click(screen.getByRole('button', { name: /tutup pengaturan/i }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
