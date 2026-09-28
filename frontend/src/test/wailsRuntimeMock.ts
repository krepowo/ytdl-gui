/**
 * Test double for the Wails runtime bindings.
 *
 * Vitest aliases `../wailsjs/runtime/runtime` (and the `@wailsjs` path) to this
 * module, so components calling window controls or subscribing to events can be
 * tested outside a Wails webview. Real Wails JS bindings don't exist in jsdom.
 */
import { vi } from 'vitest'

export const WindowMinimise = vi.fn()
export const WindowToggleMaximise = vi.fn()
export const WindowMaximise = vi.fn()
export const WindowUnmaximise = vi.fn()
export const Quit = vi.fn()
export const WindowSetTitle = vi.fn()
export const WindowCenter = vi.fn()
export const WindowShow = vi.fn()
export const WindowHide = vi.fn()
export const EventsOn = vi.fn(() => () => {})
export const EventsOff = vi.fn()
export const EventsEmit = vi.fn()
export const ClipboardGetText = vi.fn(async () => '')
export const ClipboardSetText = vi.fn(async () => true)
export const BrowserOpenURL = vi.fn()
export const Environment = vi.fn(async () => ({
  buildType: 'test',
  platform: 'windows',
  arch: 'amd64',
}))
