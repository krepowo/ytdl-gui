import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import QueueItem from './QueueItem'
import { renderWithTheme } from '../../test/renderWithTheme'
import type { Job, JobState } from '../../api/types'

function job(state: JobState, overrides: Partial<Job> = {}): Job {
  return {
    id: 'j1',
    url: 'https://example.com/v',
    title: 'My Download',
    mode: 'video',
    formatId: 'v720',
    state,
    percent: 0,
    downloadedBytes: 0,
    totalBytes: 0,
    speedBps: 0,
    etaSec: 0,
    outputPath: '',
    error: '',
    createdAt: 1,
    ...overrides,
  }
}

const noop = {
  onPause: vi.fn(),
  onResume: vi.fn(),
  onCancel: vi.fn(),
  onOpen: vi.fn(),
  onRetry: vi.fn(),
  onRemove: vi.fn(),
}

describe('QueueItem', () => {
  it('shows the title', () => {
    renderWithTheme(<QueueItem job={job('downloading')} {...noop} />)
    expect(screen.getByText('My Download')).toBeInTheDocument()
  })

  it('falls back to the URL when the title is empty', () => {
    renderWithTheme(<QueueItem job={job('queued', { title: '' })} {...noop} />)
    expect(screen.getByText('https://example.com/v')).toBeInTheDocument()
  })

  it('shows a progress bar while downloading with percent, size and speed', () => {
    renderWithTheme(
      <QueueItem
        job={job('downloading', {
          percent: 42.3,
          downloadedBytes: 42 * 1024 * 1024,
          totalBytes: 100 * 1024 * 1024,
          speedBps: 1024 * 1024,
          etaSec: 30,
        })}
        {...noop}
      />,
    )

    expect(screen.getByRole('progressbar')).toBeInTheDocument()
    const status = screen.getByText(/42\.3%/).textContent ?? ''
    expect(status).toContain('42.0 MB')
    expect(status).toContain('100.0 MB')
  })

  it('offers a pause action while downloading', async () => {
    const onPause = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<QueueItem job={job('downloading')} {...noop} onPause={onPause} />)

    await user.click(screen.getByRole('button', { name: /pause/i }))
    expect(onPause).toHaveBeenCalledTimes(1)
  })

  it('offers a resume action while paused', async () => {
    const onResume = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<QueueItem job={job('paused', { percent: 10 })} {...noop} onResume={onResume} />)

    await user.click(screen.getByRole('button', { name: /resume/i }))
    expect(onResume).toHaveBeenCalledTimes(1)
  })

  it('offers a cancel action while queued', async () => {
    const onCancel = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<QueueItem job={job('queued')} {...noop} onCancel={onCancel} />)

    await user.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('offers an open action when completed', async () => {
    const onOpen = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(
      <QueueItem job={job('completed', { percent: 100, outputPath: 'D:\\D\\clip.mp4' })} {...noop} onOpen={onOpen} />,
    )

    await user.click(screen.getByRole('button', { name: /open/i }))
    expect(onOpen).toHaveBeenCalledWith('D:\\D\\clip.mp4')
  })

  it('offers a retry action when failed and shows the error', () => {
    renderWithTheme(<QueueItem job={job('error', { error: 'network down' })} {...noop} />)

    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
    expect(screen.getByText(/network down/i)).toBeInTheDocument()
  })

  it('offers a remove action for finished jobs', async () => {
    const onRemove = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<QueueItem job={job('completed', { percent: 100 })} {...noop} onRemove={onRemove} />)

    await user.click(screen.getByRole('button', { name: /remove/i }))
    expect(onRemove).toHaveBeenCalledTimes(1)
  })

  it('shows no pause/cancel action for a canceled job', () => {
    renderWithTheme(<QueueItem job={job('canceled')} {...noop} />)

    expect(screen.queryByRole('button', { name: /pause/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument()
  })

  it('gives every icon action an accessible label', () => {
    renderWithTheme(<QueueItem job={job('downloading')} {...noop} />)
    for (const btn of screen.getAllByRole('button')) {
      expect(btn).toHaveAccessibleName()
    }
  })
})
