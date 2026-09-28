/**
 * Formatting helpers for the UI. Kept in one place so the queue item, storage
 * bar, and summary all render numbers the same way.
 */

/** Formats a byte count for humans, e.g. 1536 -> "1.5 KB". */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  let value = bytes
  let i = 0
  while (value >= base && i < units.length - 1) {
    value /= base
    i++
  }
  // Whole bytes read better without a decimal.
  return i === 0 ? `${Math.round(value)} B` : `${value.toFixed(1)} ${units[i]}`
}

/** Formats a speed in bytes/second, e.g. 1048576 -> "1.0 MB/s". */
export function formatSpeed(bytesPerSec: number): string {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec <= 0) return ''
  return `${formatBytes(bytesPerSec)}/s`
}

/**
 * Formats an ETA in seconds as mm:ss or h:mm:ss. Zero/unknown returns an empty
 * string so callers can hide it.
 */
export function formatEta(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return ''
  const total = Math.round(seconds)
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const pad = (n: number) => String(n).padStart(2, '0')
  return h > 0 ? `${pad(h)}:${pad(m)}:${pad(s)}` : `${pad(m)}:${pad(s)}`
}

/** Clamps a percentage into 0..100 for progress bars. */
export function clampPercent(percent: number): number {
  if (!Number.isFinite(percent)) return 0
  return Math.min(100, Math.max(0, percent))
}
