import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import StorageBar from './StorageBar'
import { renderWithTheme } from '../../test/renderWithTheme'
import { installWailsGoMock } from '../../test/wailsGoMock'

describe('StorageBar', () => {
  let mocks: ReturnType<typeof installWailsGoMock>

  beforeEach(() => {
    vi.clearAllMocks()
    mocks = installWailsGoMock()
  })

  it('shows the download directory path', async () => {
    mocks.GetSettings.mockResolvedValue({
      downloadDir: 'D:\\Media',
      maxConcurrent: 2,
      defaultMode: 'video',
      defaultQuality: 'best',
      filenameTemplate: '%(title)s.%(ext)s',
      cookiesBrowser: '',
    })

    renderWithTheme(<StorageBar onOpenSettings={vi.fn()} />)

    expect(await screen.findByText('D:\\Media')).toBeInTheDocument()
  })

  it('shows the free space for the download directory', async () => {
    mocks.GetDiskFree.mockResolvedValue(5 * 1024 * 1024 * 1024) // 5 GB

    renderWithTheme(<StorageBar onOpenSettings={vi.fn()} />)

    expect(await screen.findByText(/5\.0 GB/)).toBeInTheDocument()
    expect(mocks.GetDiskFree).toHaveBeenCalledWith('D:\\Downloads')
  })

  it('opens the download folder via the folder icon', async () => {
    const user = userEvent.setup()
    renderWithTheme(<StorageBar onOpenSettings={vi.fn()} />)
    await waitFor(() => expect(mocks.GetSettings).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /buka folder/i }))
    expect(mocks.OpenDownloadDir).toHaveBeenCalledTimes(1)
  })

  it('opens settings via the settings icon', async () => {
    const onOpenSettings = vi.fn()
    const user = userEvent.setup()
    renderWithTheme(<StorageBar onOpenSettings={onOpenSettings} />)
    await waitFor(() => expect(mocks.GetSettings).toHaveBeenCalled())

    await user.click(screen.getByRole('button', { name: /pengaturan/i }))
    expect(onOpenSettings).toHaveBeenCalledTimes(1)
  })

  it('gives every icon action an accessible label', async () => {
    renderWithTheme(<StorageBar onOpenSettings={vi.fn()} />)
    await waitFor(() => expect(mocks.GetSettings).toHaveBeenCalled())

    for (const btn of screen.getAllByRole('button')) {
      expect(btn).toHaveAccessibleName()
    }
  })
})
