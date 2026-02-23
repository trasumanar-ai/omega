import { useCallback, useEffect, useRef, useState } from 'react'
import { DEFAULT_GAME_CONFIG, type GameConfig } from '../game/config'
import { GameSession } from '../game/session'
import type { TickStats } from '../sim/types'

export type SimConfig = GameConfig

export function useSimulation() {
  const sessionRef = useRef(new GameSession(DEFAULT_GAME_CONFIG))
  const timerRef = useRef<number | null>(null)

  const [config, setConfigState] = useState<SimConfig>(sessionRef.current.getConfig())
  const [stats, setStats] = useState<TickStats>(sessionRef.current.getStats())
  const [running, setRunning] = useState(false)

  const getSim = useCallback(() => {
    return sessionRef.current.getSimulation()
  }, [])

  const updateConfig = useCallback((update: Partial<SimConfig> | ((prev: SimConfig) => SimConfig)) => {
    const next = sessionRef.current.setConfig(update)
    setConfigState(next)
  }, [])

  const tick = useCallback(() => {
    setStats(sessionRef.current.step())
  }, [])

  useEffect(() => {
    if (!running) {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
      return
    }

    timerRef.current = window.setInterval(tick, config.speed)
    return () => {
      if (timerRef.current !== null) {
        window.clearInterval(timerRef.current)
        timerRef.current = null
      }
    }
  }, [running, config.speed, tick])

  const start = useCallback(() => setRunning(true), [])
  const stop = useCallback(() => setRunning(false), [])
  const toggle = useCallback(() => setRunning(prev => !prev), [])

  const step = useCallback(() => {
    if (!running) {
      tick()
    }
  }, [running, tick])

  const reset = useCallback(() => {
    setRunning(false)
    setStats(sessionRef.current.reset())
    setConfigState(sessionRef.current.getConfig())
  }, [])

  return {
    sim: getSim(),
    stats,
    running,
    config,
    setConfig: updateConfig,
    start,
    stop,
    toggle,
    step,
    reset,
  }
}
