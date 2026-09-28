import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor, act } from '@testing-library/react'
import { useDownloadQueue } from './useDownloadQueue'
import { installWailsGoMock } from '../../test/wailsGoMock'
import type { Job } from '../../api/types'

// Capture the event handlers the hook registers, so tests can fire events.
type Handler = (...data: unknown[]) => void
const handlers: Record<string, Handler[]> = {}

vi.mock('../../../wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn((name: string, cb: Handler) => {
    ;(handlers[name] ||= []).push(cb)
    return () => {
      handlers[name] = (handlers[name] || []).filter((h) => h !== cb)
    }
  }),
  EventsOff: vi.fn(),
  EventsEmit: vi.fn(),
}))

function fire(name: string, ...data: unknown[]) {
  for (const h of handlers[name] || []) h(...data)
}

function job(overrides: Partial<Job> = {}): Job {
  return {
    id: 'j1',
    url: 'https://example.com/v',
    title: 'T',
    mode: 'video',
    formatId: 'v720',
    state: 'queued',
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

describe('useDownloadQueue', () => {
  let mocks: ReturnType<typeof installWailsGoMock>

  beforeEach(() => {
    for (const k of Object.keys(handlers)) delete handlers[k]
    vi.clearAllMocks()
    mocks = installWailsGoMock()
  })

  it('loads the initial job list', async () => {
    mocks.ListJobs.mockResolvedValue([job({ id: 'a' }), job({ id: 'b' })])

    const { result } = renderHook(() => useDownloadQueue())

    await waitFor(() => expect(result.current.jobs).toHaveLength(2))
    expect(result.current.jobs[0].id).toBe('a')
  })

  it('adds a job on job:added without polling', async () => {
    mocks.ListJobs.mockResolvedValue([])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(mocks.ListJobs).toHaveBeenCalled())

    act(() => fire('job:added', job({ id: 'new', title: 'New' })))

    await waitFor(() => expect(result.current.jobs).toHaveLength(1))
    expect(result.current.jobs[0].id).toBe('new')
  })

  it('updates a job on job:updated (progress without polling)', async () => {
    mocks.ListJobs.mockResolvedValue([job({ id: 'a', percent: 0, state: 'downloading' })])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(result.current.jobs).toHaveLength(1))

    act(() => fire('job:updated', job({ id: 'a', percent: 42.5, state: 'downloading' })))

    await waitFor(() => expect(result.current.jobs[0].percent).toBe(42.5))
  })

  it('removes a job on job:removed', async () => {
    mocks.ListJobs.mockResolvedValue([job({ id: 'a' }), job({ id: 'b' })])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(result.current.jobs).toHaveLength(2))

    act(() => fire('job:removed', 'a'))

    await waitFor(() => expect(result.current.jobs).toHaveLength(1))
    expect(result.current.jobs[0].id).toBe('b')
  })

  it('does not duplicate a job already present when job:added re-fires', async () => {
    mocks.ListJobs.mockResolvedValue([job({ id: 'a' })])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(result.current.jobs).toHaveLength(1))

    act(() => fire('job:added', job({ id: 'a' })))

    await waitFor(() => expect(result.current.jobs).toHaveLength(1))
  })

  it('reports counts of queued and downloading jobs', async () => {
    mocks.ListJobs.mockResolvedValue([
      job({ id: 'a', state: 'downloading' }),
      job({ id: 'b', state: 'downloading' }),
      job({ id: 'c', state: 'queued' }),
      job({ id: 'd', state: 'completed' }),
    ])
    const { result } = renderHook(() => useDownloadQueue())

    await waitFor(() => expect(result.current.jobs).toHaveLength(4))
    expect(result.current.queuedCount).toBe(1)
    expect(result.current.downloadingCount).toBe(2)
  })

  it('exposes a lastError from app:error events', async () => {
    mocks.ListJobs.mockResolvedValue([])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(mocks.ListJobs).toHaveBeenCalled())

    act(() => fire('app:error', { message: 'yt-dlp tidak ditemukan' }))

    await waitFor(() => expect(result.current.lastError).toBe('yt-dlp tidak ditemukan'))
  })

  it('unsubscribes from events on unmount', async () => {
    mocks.ListJobs.mockResolvedValue([])
    const { unmount } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(mocks.ListJobs).toHaveBeenCalled())

    const before = (handlers['job:added'] || []).length
    unmount()
    expect((handlers['job:added'] || []).length).toBe(before - 1)
  })

  it('exposes actions that call the bound methods', async () => {
    mocks.ListJobs.mockResolvedValue([job({ id: 'a', state: 'downloading' })])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(result.current.jobs).toHaveLength(1))

    await act(async () => {
      await result.current.pause('a')
      await result.current.resume('a')
      await result.current.cancel('a')
      await result.current.remove('a')
      await result.current.open('D:\\D\\clip.mp4')
    })

    expect(mocks.PauseJob).toHaveBeenCalledWith('a')
    expect(mocks.ResumeJob).toHaveBeenCalledWith('a')
    expect(mocks.CancelJob).toHaveBeenCalledWith('a')
    expect(mocks.RemoveJob).toHaveBeenCalledWith('a')
    expect(mocks.OpenFile).toHaveBeenCalledWith('D:\\D\\clip.mp4')
  })

  it('retry re-enqueues using the failed job details', async () => {
    const failed = job({ id: 'a', state: 'error', url: 'https://x/v', mode: 'audio', formatId: 'a128', title: 'T' })
    mocks.ListJobs.mockResolvedValue([failed])
    const { result } = renderHook(() => useDownloadQueue())
    await waitFor(() => expect(result.current.jobs).toHaveLength(1))

    await act(async () => {
      await result.current.retry('a')
    })

    expect(mocks.StartDownload).toHaveBeenCalledWith({
      url: 'https://x/v',
      mode: 'audio',
      formatId: 'a128',
      needsMerge: false,
      title: 'T',
    })
  })
})
