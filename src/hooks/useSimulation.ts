import { useCallback, useEffect, useRef, useState } from 'react'
import {
  DEFAULT_GAME_CONFIG,
  createInitialStats,
  normalizeGameConfig,
  toSimulationConfig,
  type GameConfig,
} from '../game/config'
import {
  RemoteSimulation,
  normalizeAgentDetail,
  type AgentDetail,
  type AgentDetailResponse,
  type SimulationStateResponse,
} from '../game/remoteClient'
import type { SimulationConfig, TickStats } from '../sim/types'

export type SimConfig = GameConfig
export type { AgentDetail }

const MAX_HISTORY = 300

export function useSimulation() {
  const simRef = useRef(new RemoteSimulation(DEFAULT_GAME_CONFIG.width, DEFAULT_GAME_CONFIG.height, DEFAULT_GAME_CONFIG.maxAgentSlots))
  const timerRef = useRef<number | null>(null)
  const historyRef = useRef<TickStats[]>([])
  const configRef = useRef<SimConfig>(DEFAULT_GAME_CONFIG)
  const pendingConfigRef = useRef(false)
  const inFlightRef = useRef(false)

  const [config, setConfigState] = useState<SimConfig>(DEFAULT_GAME_CONFIG)
  const [stats, setStats] = useState<TickStats>(createInitialStats(DEFAULT_GAME_CONFIG))
  const [statsHistory, setStatsHistory] = useState<TickStats[]>([])
  const [running, setRunning] = useState(false)

  const syncFromState = useCallback((state: SimulationStateResponse, syncConfig: boolean, resetHistory: boolean) => {
    simRef.current.applyWorld(state.world)
    setStats(state.stats)

    if (syncConfig) {
      const nextConfig = fromSimulationConfig(state.config, configRef.current)
      configRef.current = nextConfig
      setConfigState(nextConfig)
    }

    if (resetHistory) {
      historyRef.current = []
      setStatsHistory([])
    }
  }, [])

  const applyPendingConfig = useCallback(async () => {
    if (!pendingConfigRef.current) return
    const payload = toSimulationConfig(configRef.current)
    const state = await fetchJson<SimulationStateResponse>('/api/sim/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
    pendingConfigRef.current = false
    syncFromState(state, true, true)
  }, [syncFromState])

  const tick = useCallback(async () => {
    if (inFlightRef.current) return
    inFlightRef.current = true

    try {
      await applyPendingConfig()
      const state = await fetchJson<SimulationStateResponse>('/api/sim/step', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ count: 1 }),
      })
      syncFromState(state, false, false)

      const h = historyRef.current
      if (h.length >= MAX_HISTORY) h.shift()
      h.push(state.stats)
      setStatsHistory([...h])
    } catch (err) {
      console.error('Simulation tick failed:', err)
    } finally {
      inFlightRef.current = false
    }
  }, [applyPendingConfig, syncFromState])

  useEffect(() => {
    let cancelled = false

    const boot = async () => {
      try {
        const state = await fetchJson<SimulationStateResponse>('/api/sim/state')
        if (cancelled) return
        syncFromState(state, true, true)
      } catch (err) {
        console.error('Failed to load simulation state from Go backend:', err)
      }
    }

    void boot()
    return () => {
      cancelled = true
    }
  }, [syncFromState])

  useEffect(() => {
    if (!running) {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
      return
    }

    timerRef.current = window.setInterval(() => {
      void tick()
    }, config.speed)

    return () => {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [running, config.speed, tick])

  const updateConfig = useCallback((update: Partial<SimConfig> | ((prev: SimConfig) => SimConfig)) => {
    setConfigState(prev => {
      const next = typeof update === 'function'
        ? update(prev)
        : { ...prev, ...update }
      const normalized = normalizeGameConfig(next)
      configRef.current = normalized
      pendingConfigRef.current = true
      return normalized
    })
  }, [])

  const start = useCallback(() => setRunning(true), [])
  const stop = useCallback(() => setRunning(false), [])
  const toggle = useCallback(() => setRunning(prev => !prev), [])

  const step = useCallback(() => {
    if (!running) {
      void tick()
    }
  }, [running, tick])

  const reset = useCallback(() => {
    setRunning(false)

    const performReset = async () => {
      if (inFlightRef.current) return
      inFlightRef.current = true
      try {
        if (pendingConfigRef.current) {
          await applyPendingConfig()
          return
        }
        const state = await fetchJson<SimulationStateResponse>('/api/sim/reset', {
          method: 'POST',
        })
        syncFromState(state, true, true)
      } catch (err) {
        console.error('Simulation reset failed:', err)
      } finally {
        inFlightRef.current = false
      }
    }

    void performReset()
  }, [applyPendingConfig, syncFromState])

  const getAgentDetail = useCallback(async (agentId: number): Promise<AgentDetail | null> => {
    try {
      const raw = await fetchJson<AgentDetailResponse>(`/api/sim/agent/${agentId}`)
      return normalizeAgentDetail(raw)
    } catch (err) {
      console.error(`Failed to fetch agent detail for agent ${agentId}:`, err)
      return null
    }
  }, [])

  return {
    sim: simRef.current,
    stats,
    statsHistory,
    running,
    config,
    setConfig: updateConfig,
    start,
    stop,
    toggle,
    step,
    reset,
    getAgentDetail,
  }
}

async function fetchJson<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init)
  if (!response.ok) {
    const message = await response.text()
    throw new Error(`${response.status} ${response.statusText}: ${message}`)
  }
  return response.json() as Promise<T>
}

function fromSimulationConfig(input: SimulationConfig, fallback: GameConfig): GameConfig {
  const merged: GameConfig = {
    ...fallback,
    width: numberOr(input.width, fallback.width),
    height: numberOr(input.height, fallback.height),
    agents: numberOr(input.agentCount, fallback.agents),
    maxAgentSlots: numberOr(input.maxAgentSlots, fallback.maxAgentSlots),
    initialEnergy: numberOr(input.initialEnergy, fallback.initialEnergy),
    maxEnergy: numberOr(input.maxEnergy, fallback.maxEnergy),
    moveEnergyCost: numberOr(input.moveEnergyCost, fallback.moveEnergyCost),
    idleEnergyCost: numberOr(input.idleEnergyCost, fallback.idleEnergyCost),
    neuronStepEnergyCost: numberOr(input.neuronStepEnergyCost, fallback.neuronStepEnergyCost),
    minNeuronCount: numberOr(input.minNeuronCount, fallback.minNeuronCount),
    maxNeuronCount: numberOr(input.maxNeuronCount, fallback.maxNeuronCount),
    cloneEnergyThreshold: numberOr(input.cloneEnergyThreshold, fallback.cloneEnergyThreshold),
    cloneEnergyCost: numberOr(input.cloneEnergyCost, fallback.cloneEnergyCost),
    cloneChancePerTick: numberOr(input.cloneChancePerTick, fallback.cloneChancePerTick),
    speed: fallback.speed,
    treeDensity: numberOr(input.treeDensity, fallback.treeDensity),
    fruitDropPerTree: numberOr(input.fruitDropPerTree, fallback.fruitDropPerTree),
    maxFruitAroundTree: numberOr(input.maxFruitAroundTree, fallback.maxFruitAroundTree),
    maxInventory: numberOr(input.maxInventory, fallback.maxInventory),
    fruitEnergyGain: numberOr(input.fruitEnergyGain, fallback.fruitEnergyGain),
    tradeAmount: numberOr(input.tradeAmount, fallback.tradeAmount),
  }
  return normalizeGameConfig(merged)
}

function numberOr(value: number | undefined, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}
