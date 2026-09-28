import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'
import MediaSummary from './MediaSummary'
import { renderWithTheme } from '../../test/renderWithTheme'
import type { MediaInfo } from '../../api/types'

function info(overrides: Partial<MediaInfo> = {}): MediaInfo {
  return {
    id: 'abc',
    title: 'A Video',
    uploader: 'Someone',
    duration: 125,
    thumbnail: 'https://example.com/t.jpg',
    extractor: 'youtube',
    webpageUrl: 'https://example.com/v',
    isLive: false,
    videoOptions: [],
    audioOptions: [],
    ...overrides,
  }
}

describe('MediaSummary', () => {
  it('shows the title', () => {
    renderWithTheme(<MediaSummary info={info()} />)
    expect(screen.getByText('A Video')).toBeInTheDocument()
  })

  it('shows the detected source as a badge, for a non-YouTube site', () => {
    renderWithTheme(<MediaSummary info={info({ extractor: 'twitter' })} />)
    expect(screen.getByText(/twitter/i)).toBeInTheDocument()
  })

  it('shows the uploader when present', () => {
    renderWithTheme(<MediaSummary info={info({ uploader: 'Channel Nine' })} />)
    expect(screen.getByText('Channel Nine')).toBeInTheDocument()
  })

  it('formats a duration as mm:ss', () => {
    renderWithTheme(<MediaSummary info={info({ duration: 125 })} />)
    expect(screen.getByText('02:05')).toBeInTheDocument()
  })

  it('formats a long duration as h:mm:ss', () => {
    renderWithTheme(<MediaSummary info={info({ duration: 3725 })} />)
    expect(screen.getByText('01:02:05')).toBeInTheDocument()
  })

  it('renders without a thumbnail when the site provides none', () => {
    renderWithTheme(<MediaSummary info={info({ thumbnail: '' })} />)
    // No broken image element is rendered.
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    // The title still shows, so the summary is still useful.
    expect(screen.getByText('A Video')).toBeInTheDocument()
  })

  it('hides the duration when the site reports none', () => {
    renderWithTheme(<MediaSummary info={info({ duration: 0 })} />)
    expect(screen.queryByText(/^\d{2}:\d{2}$/)).not.toBeInTheDocument()
  })

  it('hides the uploader when absent (never shows "undefined")', () => {
    renderWithTheme(<MediaSummary info={info({ uploader: '' })} />)
    expect(screen.queryByText(/undefined/i)).not.toBeInTheDocument()
  })

  it('shows a LIVE badge for a live stream', () => {
    renderWithTheme(<MediaSummary info={info({ isLive: true })} />)
    expect(screen.getByText(/live/i)).toBeInTheDocument()
  })

  it('uses the thumbnail as the image alt text derived from the title', () => {
    renderWithTheme(<MediaSummary info={info({ title: 'My Clip' })} />)
    expect(screen.getByRole('img')).toHaveAttribute('alt', 'My Clip')
  })
})
