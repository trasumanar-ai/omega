import { useCallback, useEffect, useRef, useState } from 'react'
import { Simulation } from '../sim/simulation'
import { emptyActionHistogram, type TickStats } from '../sim/types'

export type SimConfig = {
  width: number
  height: number
  agents: number
  speed: number
  initialEnergy: number
  moveEnergyCost: number
  idleEnergyCost: number
}

const DEFAULTS: SimConfig = {
  width: 80,
  height: 50,
  agents: 800,
  speed: 60,
  initialEnergy: 80,
  moveEnergyCost: 1,
  idleEnergyCost: 0.25,
}

export function useSimulation() {
  const [config, setConfig] = useState<SimConfig>(DEFAULTS)
  const [stats, setStats] = useState<TickStats>({
    tick: 0,
    movedAgents: 0,
    occupancy: 0,
    aliveAgents: DEFAULTS.agents,
    deathsThisTick: 0,
    avgEnergy: DEFAULTS.initialEnergy,
    actionHistogram: emptyActionHistogram(),
  })
  const [running, setRunning] = useState(false)

  const simRef = useRef<Simulation | null>(null)
  const timerRef = useRef<number | null>(null)
  const dirtyRef = useRef(true) // sim needs rebuild

  const buildSim = useCallback(() => {
    const w = clamp(config.width, 8, 200)
    const h = clamp(config.height, 8, 200)
    const a = Math.min(clamp(config.agents, 1, 30000), w * h)
    const initialEnergy = clamp(config.initialEnergy, 1, 500)
    const moveEnergyCost = clamp(config.moveEnergyCost, 0.05, 20)
    const idleEnergyCost = clamp(config.idleEnergyCost, 0.01, 10)
    simRef.current = new Simulation(
      {
        width: w,
        height: h,
        agentCount: a,
        initialEnergy,
        moveEnergyCost,
        idleEnergyCost,
      },
      Date.now() & 0xffffffff,
    )
    setStats({
      tick: 0,
      movedAgents: 0,
      occupancy: a / (w * h),
      aliveAgents: a,
      deathsThisTick: 0,
      avgEnergy: initialEnergy,
      actionHistogram: emptyActionHistogram(),
    })
    dirtyRef.current = false
    return simRef.current
  }, [config])

  const getSim = useCallback(() => {
    if (!simRef.current || dirtyRef.current) return buildSim()
    return simRef.current
  }, [buildSim])

  const tick = useCallback(() => {
    const sim = getSim()
    setStats(sim.step())
  }, [getSim])

  // mark dirty when config changes
  const updateConfig = useCallback((update: Partial<SimConfig> | ((prev: SimConfig) => SimConfig)) => {
    setConfig(prev => {
      const next = typeof update === 'function' ? update(prev) : { ...prev, ...update }
      dirtyRef.current = true
      return next
    })
  }, [])

  // loop
  useEffect(() => {
    if (!running) return
    timerRef.current = window.setInterval(tick, config.speed)
    return () => { if (timerRef.current) clearInterval(timerRef.current) }
  }, [running, config.speed, tick])

  const start = useCallback(() => setRunning(true), [])
  const stop = useCallback(() => setRunning(false), [])
  const toggle = useCallback(() => setRunning(r => !r), [])
  const step = useCallback(() => { if (!running) tick() }, [running, tick])

  const reset = useCallback(() => {
    setRunning(false)
    dirtyRef.current = true
    simRef.current = null
    setStats({
      tick: 0,
      movedAgents: 0,
      occupancy: 0,
      aliveAgents: config.agents,
      deathsThisTick: 0,
      avgEnergy: config.initialEnergy,
      actionHistogram: emptyActionHistogram(),
    })
  }, [config.agents, config.initialEnergy])

  return {
    sim: getSim(),
    stats,
    running,
    config,
    setConfig: updateConfig,
    start, stop, toggle, step, reset,
  }
}

function clamp(v: number, lo: number, hi: number) {
  if (!Number.isFinite(v)) return lo
  return Math.max(lo, Math.min(hi, v))
}
