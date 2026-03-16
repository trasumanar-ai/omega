import { useCallback, useEffect, useRef, useState } from 'react'
import type { EconConfig, EconExperimentInfo, EconExperimentRecord, EconLabArchiveRecord, EconLabArchiveSummary, EconLabResponse, EconRunInfo, EconRunRecord, EconStateResponse, MarketSnapshot, TickStats, WorldSnapshot } from '../econ/types'

type BackendStatus = {
  connected: boolean
  error: string | null
}

const MAX_HISTORY = 240
const RECONNECT_MS = 2000

export function useEconSimulation() {
  const [world, setWorld] = useState<WorldSnapshot | null>(null)
  const [config, setConfig] = useState<EconConfig | null>(null)
  const [market, setMarket] = useState<MarketSnapshot | null>(null)
  const [stats, setStats] = useState<TickStats | null>(null)
  const [statsHistory, setStatsHistory] = useState<TickStats[]>([])
  const [recentEvents, setRecentEvents] = useState<string[]>([])
  const [currentRun, setCurrentRun] = useState<EconRunInfo | null>(null)
  const [runs, setRuns] = useState<EconRunInfo[]>([])
  const [experiments, setExperiments] = useState<EconExperimentInfo[]>([])
  const [lab, setLab] = useState<EconLabResponse | null>(null)
  const [labArchive, setLabArchive] = useState<EconLabArchiveSummary[]>([])
  const [labRunningAll, setLabRunningAll] = useState(false)
  const [labRunningSessionId, setLabRunningSessionId] = useState<string | null>(null)
  const [running, setRunning] = useState(false)
  const [speed, setSpeed] = useState(650)
  const [llmMode, setLlmMode] = useState('unknown')
  const [llmModel, setLlmModel] = useState('')
  const [backendStatus, setBackendStatus] = useState<BackendStatus>({
    connected: false,
    error: null,
  })

  const timerRef = useRef<number | null>(null)
  const labTimerRef = useRef<number | null>(null)
  const historyRef = useRef<TickStats[]>([])
  const inFlightRef = useRef(false)

  const sync = useCallback((state: EconStateResponse, resetHistory: boolean) => {
    setConfig(state.config)
    setWorld(state.world)
    setMarket(state.market)
    setStats(state.stats)
    setRecentEvents(state.recentEvents)
    setCurrentRun(state.run)
    setLlmMode(state.llmMode)
    setLlmModel(state.llmModel)
    if (resetHistory) {
      historyRef.current = state.stats.tick > 0 ? [state.stats] : []
      setStatsHistory(historyRef.current)
      return
    }

    const next = historyRef.current.slice(-MAX_HISTORY + 1)
    next.push(state.stats)
    historyRef.current = next
    setStatsHistory(next)
  }, [])

  const loadState = useCallback(async (resetHistory: boolean) => {
    const state = await fetchJson<EconStateResponse>('/api/econ/state')
    sync(state, resetHistory)
    setBackendStatus({ connected: true, error: null })
  }, [sync])

  const loadRuns = useCallback(async () => {
    try {
      const nextRuns = await fetchJson<EconRunInfo[]>('/api/econ/runs')
      setRuns(nextRuns)
    } catch {
      // Keep the current list if run history is temporarily unavailable.
    }
  }, [])

  const loadRunRecord = useCallback(async (runId: string) => {
    return fetchJson<EconRunRecord>(`/api/econ/runs/${runId}`)
  }, [])

  const loadExperiments = useCallback(async () => {
    const result = await fetchJson<EconExperimentInfo[]>('/api/econ/experiments')
    setExperiments(result)
    return result
  }, [])

  const loadExperimentRecord = useCallback(async (experimentId: string) => {
    return fetchJson<EconExperimentRecord>(`/api/econ/experiments/${experimentId}`)
  }, [])

  const loadLab = useCallback(async () => {
    const result = await fetchJson<EconLabResponse>('/api/econ/lab')
    setLab(result)
    return result
  }, [])

  const loadLabArchive = useCallback(async () => {
    const result = await fetchJson<EconLabArchiveSummary[]>('/api/econ/lab/archive')
    setLabArchive(result)
    return result
  }, [])

  const loadLabArchiveRecord = useCallback(async (archiveId: string) => {
    return fetchJson<EconLabArchiveRecord>(`/api/econ/lab/archive/${archiveId}`)
  }, [])

  const stepAllLab = useCallback(async (count = 1) => {
    const result = await fetchJson<EconLabResponse>('/api/econ/lab/step', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ count }),
    })
    setLab(result)
    return result
  }, [])

  const resetAllLab = useCallback(async () => {
    setLabRunningAll(false)
    setLabRunningSessionId(null)
    const result = await fetchJson<EconLabResponse>('/api/econ/lab/reset', {
      method: 'POST',
    })
    setLab(result)
    return result
  }, [])

  const createLabSession = useCallback(async (payload?: { name?: string; presets?: string[]; algorithms?: string[] }) => {
    const result = await fetchJson<EconLabResponse>('/api/econ/lab/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload ?? {}),
    })
    setLab(result)
    return result
  }, [])

  const stepLabSession = useCallback(async (sessionId: string, count = 1) => {
    const result = await fetchJson<EconLabResponse>(`/api/econ/lab/sessions/${sessionId}/step`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ count }),
    })
    setLab(result)
    return result
  }, [])

  const resetLabSession = useCallback(async (sessionId: string, payload?: { presets?: string[]; algorithms?: string[] }) => {
    setLabRunningSessionId(current => (current === sessionId ? null : current))
    const result = await fetchJson<EconLabResponse>(`/api/econ/lab/sessions/${sessionId}/reset`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload ?? {}),
    })
    setLab(result)
    return result
  }, [])

  const deleteLabSession = useCallback(async (sessionId: string) => {
    setLabRunningSessionId(current => (current === sessionId ? null : current))
    const result = await fetchJson<EconLabResponse>(`/api/econ/lab/sessions/${sessionId}`, {
      method: 'DELETE',
    })
    setLab(result)
    return result
  }, [])

  const saveLabSession = useCallback(async (sessionId: string, payload?: { name?: string; notes?: string }) => {
    await fetchJson<{ ok: boolean }>(`/api/econ/lab/sessions/${sessionId}/save`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload ?? {}),
    })
    return loadLabArchive()
  }, [loadLabArchive])

  const step = useCallback(async (count = 1) => {
    if (inFlightRef.current) return
    inFlightRef.current = true
    try {
      const state = await fetchJson<EconStateResponse>('/api/econ/step', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ count }),
      })
      sync(state, false)
      if (state.stats.tick === 1 || state.stats.tick % 12 === 0) {
        void loadRuns()
      }
      setBackendStatus({ connected: true, error: null })
    } catch (err) {
      setBackendStatus({
        connected: false,
        error: err instanceof Error ? err.message : 'Unknown backend error',
      })
    } finally {
      inFlightRef.current = false
    }
  }, [loadRuns, sync])

  const reset = useCallback(async () => {
    setRunning(false)
    if (inFlightRef.current) return
    inFlightRef.current = true
    try {
      const state = await fetchJson<EconStateResponse>('/api/econ/reset', {
        method: 'POST',
      })
      sync(state, true)
      void loadRuns()
      setBackendStatus({ connected: true, error: null })
    } catch (err) {
      setBackendStatus({
        connected: false,
        error: err instanceof Error ? err.message : 'Unknown backend error',
      })
    } finally {
      inFlightRef.current = false
    }
  }, [loadRuns, sync])

  useEffect(() => {
    void loadState(true)
    void loadRuns()
    void loadExperiments()
    void loadLab()
    void loadLabArchive()
  }, [loadExperiments, loadLab, loadLabArchive, loadRuns, loadState])

  useEffect(() => {
    if (!labRunningSessionId || !lab) return
    if (!lab.sessions.some(session => session.id === labRunningSessionId)) {
      setLabRunningSessionId(null)
    }
  }, [lab, labRunningSessionId])

  useEffect(() => {
    if (backendStatus.connected) return

    const timer = window.setInterval(() => {
      void loadState(false)
    }, RECONNECT_MS)

    return () => {
      window.clearInterval(timer)
    }
  }, [backendStatus.connected, loadState])

  useEffect(() => {
    if (!running) {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
      return
    }

    timerRef.current = window.setInterval(() => {
      void step()
    }, speed)

    return () => {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [running, speed, step])

  useEffect(() => {
    if (!labRunningAll && !labRunningSessionId) {
      if (labTimerRef.current !== null) {
        window.clearInterval(labTimerRef.current)
        labTimerRef.current = null
      }
      return
    }

    labTimerRef.current = window.setInterval(() => {
      if (labRunningAll) {
        void stepAllLab()
        return
      }
      if (labRunningSessionId) {
        void stepLabSession(labRunningSessionId)
      }
    }, speed)

    return () => {
      if (labTimerRef.current !== null) {
        window.clearInterval(labTimerRef.current)
        labTimerRef.current = null
      }
    }
  }, [labRunningAll, labRunningSessionId, speed, stepAllLab, stepLabSession])

  return {
    world,
    config,
    market,
    stats,
    statsHistory,
    recentEvents,
    currentRun,
    runs,
    experiments,
    lab,
    labArchive,
    labRunningAll,
    labRunningSessionId,
    running,
    speed,
    setSpeed,
    llmMode,
    llmModel,
    backendStatus,
    start: () => setRunning(true),
    stop: () => setRunning(false),
    toggle: () => setRunning(prev => !prev),
    step,
    reset,
    startAllLab: () => {
      setLabRunningSessionId(null)
      setLabRunningAll(true)
    },
    stopLab: () => {
      setLabRunningAll(false)
      setLabRunningSessionId(null)
    },
    toggleAllLab: () => {
      setLabRunningSessionId(null)
      setLabRunningAll(prev => !prev)
    },
    startLabSession: (sessionId: string) => {
      setLabRunningAll(false)
      setLabRunningSessionId(sessionId)
    },
    toggleLabSession: (sessionId: string) => {
      setLabRunningAll(false)
      setLabRunningSessionId(current => (current === sessionId ? null : sessionId))
    },
    stepAllLab,
    resetAllLab,
    createLabSession,
    stepLabSession,
    resetLabSession,
    deleteLabSession,
    saveLabSession,
    loadRuns,
    loadRunRecord,
    loadExperiments,
    loadExperimentRecord,
    loadLab,
    loadLabArchive,
    loadLabArchiveRecord,
    isLabSessionRunning: (sessionId: string) => labRunningSessionId === sessionId,
  }
}

async function fetchJson<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  if (!res.ok) {
    const text = await res.text()
    throw new Error(`HTTP ${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}
