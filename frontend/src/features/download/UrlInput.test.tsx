import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import UrlInput from './UrlInput'
import { renderWithTheme } from '../../test/renderWithTheme'

describe('UrlInput', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('disables the Download button while the URL is empty', () => {
    renderWithTheme(<UrlInput value="" onChange={vi.fn()} onDownload={vi.fn()} />)

    expect(screen.getByRole('button', { name: /download/i })).toBeDisabled()
  })

  it('disables the Download button when the URL is only whitespace', () => {
    renderWithTheme(<UrlInput value="   " onChange={vi.fn()} onDownload={vi.fn()} />)

    expect(screen.getByRole('button', { name: /download/i })).toBeDisabled()
  })

  it('enables the Download button for a non-YouTube URL', () => {
    renderWithTheme(
      <UrlInput value="https://vimeo.com/12345" onChange={vi.fn()} onDownload={vi.fn()} />,
    )

    expect(screen.getByRole('button', { name: /download/i })).toBeEnabled()
  })

  it('calls onChange as the user types', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<UrlInput value="" onChange={onChange} onDownload={vi.fn()} />)

    await user.type(screen.getByRole('textbox'), 'abc')

    expect(onChange).toHaveBeenCalled()
  })

  it('calls onDownload when the button is clicked', async () => {
    const onDownload = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(
      <UrlInput value="https://example.com/v" onChange={vi.fn()} onDownload={onDownload} />,
    )

    await user.click(screen.getByRole('button', { name: /download/i }))

    expect(onDownload).toHaveBeenCalledTimes(1)
  })

  it('shows a spinner and disables the button while probing', () => {
    renderWithTheme(
      <UrlInput
        value="https://example.com/v"
        onChange={vi.fn()}
        onDownload={vi.fn()}
        probing
      />,
    )

    expect(screen.getByRole('progressbar')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /download/i })).toBeDisabled()
  })

  it('submits on Enter so the flow is keyboard reachable', async () => {
    const onDownload = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(
      <UrlInput value="https://example.com/v" onChange={vi.fn()} onDownload={onDownload} />,
    )

    await user.type(screen.getByRole('textbox'), '{Enter}')

    expect(onDownload).toHaveBeenCalledTimes(1)
  })

  it('uses a source-agnostic placeholder (no YouTube-only wording)', () => {
    renderWithTheme(<UrlInput value="" onChange={vi.fn()} onDownload={vi.fn()} />)

    const input = screen.getByRole('textbox')
    expect(input).toHaveAttribute('placeholder')
    expect(input.getAttribute('placeholder')?.toLowerCase()).not.toContain('youtube')
  })
})
