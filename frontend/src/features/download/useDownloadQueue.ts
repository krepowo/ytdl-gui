import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { EventsOn } from '../../../wailsjs/runtime/runtime'
import { api, errorMessage } from '../../api/client'
import type { DownloadRequest, Job } from '../../api/types'

/**
 * Subscribes to the backend's job events and exposes the live queue plus the
 * per-item actions.
 *
 * State is driven entirely by events (`job:added/updated/removed`, `app:error`);
 * the initial list is fetched once on mount. There is deliberately no polling
 * timer — the backend throttles `job:updated` to 4/s per job.
 */
export function useDownloadQueue() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [lastError, setLastError] = useState('')
  // Keep the latest list for retry lookups without re-subscribing.
  const jobsRef = useRef<Job[]>([])
  useEffect(() => {
    jobsRef.current = jobs
  }, [jobs])

  useEffect(() => {
    let active = true

    // Initial snapshot (also restores history from a previous session).
    api
      .listJobs()
      .then((initial) => {
        if (active) setJobs(initial)
      })
      .catch((err) => {
        if (active) setLastError(errorMessage(err))
      })

    const offAdded = EventsOn('job:added', (job: Job) => {
      setJobs((prev) => (prev.some((j) => j.id === job.id) ? prev : [...prev, job]))
    })

    const offUpdated = EventsOn('job:updated', (job: Job) => {
      setJobs((prev) => {
        const idx = prev.findIndex((j) => j.id === job.id)
        if (idx === -1) return [...prev, job]
        const next = prev.slice()
        next[idx] = job
        return next
      })
    })

    const offRemoved = EventsOn('job:removed', (id: string) => {
      setJobs((prev) => prev.filter((j) => j.id !== id))
    })

    const offError = EventsOn('app:error', (payload: { message?: string }) => {
      setLastError(payload?.message ?? 'Terjadi kesalahan.')
    })

    return () => {
      active = false
      offAdded()
      offUpdated()
      offRemoved()
      offError()
    }
  }, [])

  const pause = useCallback(async (id: string) => {
    try {
      await api.pauseJob(id)
    } catch (err) {
      setLastError(errorMessage(err))
    }
  }, [])

  const resume = useCallback(async (id: string) => {
    try {
      await api.resumeJob(id)
    } catch (err) {
      setLastError(errorMessage(err))
    }
  }, [])

  const cancel = useCallback(async (id: string) => {
    try {
      await api.cancelJob(id)
    } catch (err) {
      setLastError(errorMessage(err))
    }
  }, [])

  const remove = useCallback(async (id: string) => {
    try {
      await api.removeJob(id)
    } catch (err) {
      setLastError(errorMessage(err))
    }
  }, [])

  const open = useCallback(async (path: string) => {
    if (!path) return
    try {
      await api.openFile(path)
    } catch (err) {
      setLastError(errorMessage(err))
    }
  }, [])

  /** Re-enqueues a failed job using its original request details. */
  const retry = useCallback(async (id: string) => {
    const job = jobsRef.current.find((j) => j.id === id)
    if (!job) return
    const req: DownloadRequest = {
      url: job.url,
      mode: job.mode,
      formatId: job.formatId,
      needsMerge: false,
      title: job.title,
    }
    try {
      await api.startDownload(req)
    } catch (err) {
      setLastError(errorMessage(err))
    }
  }, [])

  const { queuedCount, downloadingCount } = useMemo(() => {
    let queued = 0
    let downloading = 0
    for (const j of jobs) {
      if (j.state === 'queued') queued++
      else if (j.state === 'downloading') downloading++
    }
    return { queuedCount: queued, downloadingCount: downloading }
  }, [jobs])

  return {
    jobs,
    queuedCount,
    downloadingCount,
    lastError,
    clearError: () => setLastError(''),
    pause,
    resume,
    cancel,
    remove,
    open,
    retry,
  }
}
