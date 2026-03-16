import { useCallback, useState } from 'react'
import type { SimulationConfig } from '../sim/types'

export type RunSummary = {
  id: string
  seed: number
  startedAt: string
  endedAt: string
  reason: string
  stepCount: number
  maxTick: number
  finalTick: number
  finalAlive: number
  finalEnergy: number
}

export type RunTracePoint = {
  tick: number
  aliveAgents: number
  avgEnergy: number
  avgInventory: number
  deathsThisTick: number
  collectedFruit: number
  eatenFruit: number
  tradedFruit: number
  groundFruitTotal: number
  deficientAgents: number
}

export type RunDetail = {
  id: string
  seed: number
  startedAt: string
  endedAt: string
  reason: string
  config: SimulationConfig
  stepCount: number
  maxTick: number
  final: {
    tick: number
    aliveAgents: number
    avgEnergy: number
  }
  trace: RunTracePoint[]
}

type RunsResponse = {
  directory: string
  activeRun: RunSummary | null
  runs: RunSummary[]
}

export function useRunHistory() {
  const [runs, setRuns] = useState<RunSummary[]>([])
  const [selectedRun, setSelectedRun] = useState<RunDetail | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await fetch('/api/sim/runs')
      if (!resp.ok) return
      const data: RunsResponse = await resp.json()
      setRuns(data.runs ?? [])
    } catch {
      // ignore fetch errors
    } finally {
      setLoading(false)
    }
  }, [])

  const selectRun = useCallback(async (id: string) => {
    setLoading(true)
    try {
      const resp = await fetch(`/api/sim/runs/${id}`)
      if (!resp.ok) return
      const data: RunDetail = await resp.json()
      setSelectedRun(data)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }, [])

  const clearSelection = useCallback(() => {
    setSelectedRun(null)
  }, [])

  return { runs, selectedRun, loading, refresh, selectRun, clearSelection }
}
