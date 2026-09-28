import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import QueueList from './QueueList'
import { renderWithTheme } from '../../test/renderWithTheme'
import type { Job } from '../../api/types'

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

const actions = {
  onPause: vi.fn(),
  onResume: vi.fn(),
  onCancel: vi.fn(),
  onOpen: vi.fn(),
  onRetry: vi.fn(),
  onRemove: vi.fn(),
}

describe('QueueList', () => {
  beforeEach(() => vi.clearAllMocks())

  it('shows the "Antrean Download" header', () => {
    renderWithTheme(<QueueList jobs={[]} {...actions} />)
    expect(screen.getByText(/antrean download/i)).toBeInTheDocument()
  })

  it('shows an empty state when there are no jobs', () => {
    renderWithTheme(<QueueList jobs={[]} {...actions} />)
    expect(screen.getByText('Belum ada unduhan.')).toBeInTheDocument()
  })

  it('shows the queued and downloading counts in the header', () => {
    renderWithTheme(
      <QueueList
        jobs={[
          job({ id: 'a', state: 'queued' }),
          job({ id: 'b', state: 'queued' }),
          job({ id: 'c', state: 'downloading' }),
          job({ id: 'd', state: 'completed' }),
        ]}
        {...actions}
      />,
    )

    // "2 antrean • 1 sedang diunduh"
    expect(screen.getByText(/2 antrean/)).toBeInTheDocument()
    expect(screen.getByText(/1 sedang diunduh/)).toBeInTheDocument()
  })

  it('renders one item per job', () => {
    renderWithTheme(
      <QueueList
        jobs={[job({ id: 'a', title: 'First' }), job({ id: 'b', title: 'Second' })]}
        {...actions}
      />,
    )

    expect(screen.getByText('First')).toBeInTheDocument()
    expect(screen.getByText('Second')).toBeInTheDocument()
  })

  it('does not show the empty state when jobs exist', () => {
    renderWithTheme(<QueueList jobs={[job({ id: 'a' })]} {...actions} />)
    expect(screen.queryByText('Belum ada unduhan.')).not.toBeInTheDocument()
  })
})
