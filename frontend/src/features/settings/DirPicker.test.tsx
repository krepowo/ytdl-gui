import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DirPicker from './DirPicker'
import ConfigPathNote from './ConfigPathNote'
import { renderWithTheme } from '../../test/renderWithTheme'
import { installWailsGoMock } from '../../test/wailsGoMock'

describe('DirPicker', () => {
  let mocks: ReturnType<typeof installWailsGoMock>

  beforeEach(() => {
    vi.clearAllMocks()
    mocks = installWailsGoMock()
  })

  it('shows the current directory value', () => {
    renderWithTheme(<DirPicker value={'D:\\Media'} onChange={vi.fn()} />)
    expect(screen.getByRole('textbox', { name: 'Folder unduhan' })).toHaveValue('D:\\Media')
  })

  it('calls PickDownloadDir and reports the chosen folder', async () => {
    mocks.PickDownloadDir.mockResolvedValue('E:\\New Folder')
    const onChange = vi.fn()
    const user = userEvent.setup()

    renderWithTheme(<DirPicker value={'D:\\Media'} onChange={onChange} />)
    await user.click(screen.getByRole('button', { name: /pilih folder/i }))

    expect(mocks.PickDownloadDir).toHaveBeenCalledTimes(1)
    expect(onChange).toHaveBeenCalledWith('E:\\New Folder')
  })

  it('does not change the value when the picker is cancelled', async () => {
    mocks.PickDownloadDir.mockResolvedValue('')
    const onChange = vi.fn()
    const user = userEvent.setup()

    renderWithTheme(<DirPicker value={'D:\\Media'} onChange={onChange} />)
    await user.click(screen.getByRole('button', { name: /pilih folder/i }))

    expect(onChange).not.toHaveBeenCalled()
  })
})

describe('ConfigPathNote', () => {
  let mocks: ReturnType<typeof installWailsGoMock>

  beforeEach(() => {
    vi.clearAllMocks()
    mocks = installWailsGoMock()
  })

  it('renders the active config path from GetAppInfo', async () => {
    mocks.GetAppInfo.mockResolvedValue({
      version: '0.1.0',
      configPath: 'C:\\Program Files\\Video Downloader\\config.json',
      ytDlpVersion: '2026.08.19',
      ffmpegVersion: '9.0',
    })

    renderWithTheme(<ConfigPathNote />)

    expect(
      await screen.findByText(/C:\\Program Files\\Video Downloader\\config\.json/),
    ).toBeInTheDocument()
  })

  it('shows the app and bundled binary versions', async () => {
    mocks.GetAppInfo.mockResolvedValue({
      version: '0.1.0',
      configPath: 'C:\\app\\config.json',
      ytDlpVersion: '2026.08.19',
      ffmpegVersion: '9.0',
    })

    renderWithTheme(<ConfigPathNote />)

    expect(await screen.findByText(/2026\.08\.19/)).toBeInTheDocument()
    expect(screen.getByText(/9\.0/)).toBeInTheDocument()
  })
})
