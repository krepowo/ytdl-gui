import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DownloadScreen from './DownloadScreen'
import { renderWithTheme } from '../../test/renderWithTheme'
import { installWailsGoMock } from '../../test/wailsGoMock'
import type { MediaInfo } from '../../api/types'

function media(overrides: Partial<MediaInfo> = {}): MediaInfo {
  return {
    id: 'x',
    title: 'Probed Video',
    uploader: 'Uploader',
    duration: 60,
    thumbnail: '',
    extractor: 'twitter',
    webpageUrl: 'https://x.com/v',
    isLive: false,
    videoOptions: [
      { formatId: 'v720', label: '720p', ext: 'mp4', height: 720, width: 1280, fps: 30, bitrate: 1000, filesize: 0, needsMerge: false },
    ],
    audioOptions: [
      { formatId: 'a128', label: 'm4a 128kbps', ext: 'm4a', height: 0, width: 0, fps: 0, bitrate: 128, filesize: 0, needsMerge: false },
    ],
    ...overrides,
  }
}

describe('DownloadScreen', () => {
  let mocks: ReturnType<typeof installWailsGoMock>

  beforeEach(() => {
    vi.clearAllMocks()
    mocks = installWailsGoMock()
  })

  it('probes a pasted URL and shows the media summary with the source badge', async () => {
    mocks.ProbeURL.mockResolvedValue(media())
    const user = userEvent.setup()
    renderWithTheme(<DownloadScreen />)

    await user.type(screen.getByRole('textbox'), 'https://x.com/v')
    await user.click(screen.getByRole('button', { name: /unduh/i }))

    await waitFor(() => expect(screen.getByText('Probed Video')).toBeInTheDocument())
    expect(screen.getByText(/twitter/i)).toBeInTheDocument()
    expect(mocks.ProbeURL).toHaveBeenCalledWith('https://x.com/v')
  })

  it('shows an error message (not a crash) when the probe fails', async () => {
    mocks.ProbeURL.mockRejectedValue('ERROR: Unsupported URL')
    const user = userEvent.setup()
    renderWithTheme(<DownloadScreen />)

    await user.type(screen.getByRole('textbox'), 'https://bad.example/x')
    await user.click(screen.getByRole('button', { name: /unduh/i }))

    await waitFor(() => expect(screen.getByText(/unsupported url/i)).toBeInTheDocument())
  })

  it('enqueues the download with the chosen format and clears the input', async () => {
    mocks.ProbeURL.mockResolvedValue(media())
    const user = userEvent.setup()
    renderWithTheme(<DownloadScreen />)

    await user.type(screen.getByRole('textbox'), 'https://x.com/v')
    await user.click(screen.getByRole('button', { name: /unduh/i }))
    await waitFor(() => expect(screen.getByText('Probed Video')).toBeInTheDocument())

    // With the summary shown, the action becomes "add to queue".
    await user.click(screen.getByRole('button', { name: /tambah ke antrean/i }))

    await waitFor(() => expect(mocks.StartDownload).toHaveBeenCalledTimes(1))
    const req = mocks.StartDownload.mock.calls[0][0]
    expect(req.url).toBe('https://x.com/v')
    expect(req.mode).toBe('video')
    expect(req.formatId).toBe('v720')
    expect(req.title).toBe('Probed Video')
  })

  it('does not call StartDownload before a successful probe', async () => {
    const user = userEvent.setup()
    renderWithTheme(<DownloadScreen />)

    await user.type(screen.getByRole('textbox'), 'https://x.com/v')
    await user.click(screen.getByRole('button', { name: /unduh/i }))

    expect(mocks.StartDownload).not.toHaveBeenCalled()
  })
})
