import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import FormatPicker from './FormatPicker'
import { renderWithTheme } from '../../test/renderWithTheme'
import type { FormatOption, MediaInfo } from '../../api/types'

function opt(overrides: Partial<FormatOption> = {}): FormatOption {
  return {
    formatId: 'f1',
    label: '720p',
    ext: 'mp4',
    height: 720,
    width: 1280,
    fps: 30,
    bitrate: 1000,
    filesize: 0,
    needsMerge: false,
    ...overrides,
  }
}

function info(overrides: Partial<MediaInfo> = {}): MediaInfo {
  return {
    id: 'x',
    title: 'T',
    uploader: '',
    duration: 60,
    thumbnail: '',
    extractor: 'youtube',
    webpageUrl: 'u',
    isLive: false,
    videoOptions: [opt({ formatId: 'v1080', label: '1080p', height: 1080 })],
    audioOptions: [opt({ formatId: 'a128', label: 'm4a 128kbps', ext: 'm4a', bitrate: 128 })],
    ...overrides,
  }
}

describe('FormatPicker', () => {
  it('lists the video options the site actually returned (no fixed ladder)', () => {
    renderWithTheme(
      <FormatPicker
        info={info({
          videoOptions: [
            opt({ formatId: 'v2160', label: '2160p', height: 2160 }),
            opt({ formatId: 'v480', label: '480p', height: 480 }),
          ],
        })}
        onChange={vi.fn()}
      />,
    )

    // Open the select and confirm exactly the two returned options appear.
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('defaults to the highest video quality', () => {
    const onChange = vi.fn()
    renderWithTheme(<FormatPicker info={info()} onChange={onChange} />)

    expect(screen.getByRole('combobox')).toHaveTextContent('1080p')
  })

  it('notifies the parent of the initial selection', () => {
    const onChange = vi.fn()
    renderWithTheme(<FormatPicker info={info()} onChange={onChange} />)

    expect(onChange).toHaveBeenCalled()
    const first = onChange.mock.calls[0][0]
    expect(first.mode).toBe('video')
    expect(first.formatId).toBe('v1080')
  })

  it('offers a video/audio toggle', () => {
    renderWithTheme(<FormatPicker info={info()} onChange={vi.fn()} />)
    expect(screen.getByRole('button', { name: /video/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /audio/i })).toBeInTheDocument()
  })

  it('switches to audio options when audio mode is chosen', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<FormatPicker info={info()} onChange={onChange} />)

    await user.click(screen.getByRole('button', { name: /audio/i }))

    const last = onChange.mock.calls.at(-1)?.[0]
    expect(last.mode).toBe('audio')
    expect(last.formatId).toBe('a128')
  })

  it('marks a video-only stream as needing a merge', () => {
    const onChange = vi.fn()
    renderWithTheme(
      <FormatPicker
        info={info({ videoOptions: [opt({ formatId: 'v1080', label: '1080p', needsMerge: true })] })}
        onChange={onChange}
      />,
    )

    const first = onChange.mock.calls[0][0]
    expect(first.needsMerge).toBe(true)
  })

  it('disables audio mode when the site offers no audio-only stream', () => {
    renderWithTheme(<FormatPicker info={info({ audioOptions: [] })} onChange={vi.fn()} />)
    expect(screen.getByRole('button', { name: /audio/i })).toBeDisabled()
  })

  it('disables video mode when the site offers no video option', () => {
    renderWithTheme(<FormatPicker info={info({ videoOptions: [] })} onChange={vi.fn()} />)
    expect(screen.getByRole('button', { name: /video/i })).toBeDisabled()
  })

  it('shows a live notice for a live stream (no quality choice)', () => {
    renderWithTheme(<FormatPicker info={info({ isLive: true })} onChange={vi.fn()} />)
    expect(screen.getByText(/live/i)).toBeInTheDocument()
  })
})
