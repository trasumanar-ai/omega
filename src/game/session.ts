import { Simulation } from '../sim/simulation'
import type { TickStats } from '../sim/types'
import {
  DEFAULT_GAME_CONFIG,
  createInitialStats,
  normalizeGameConfig,
  toSimulationConfig,
  type GameConfig,
} from './config'
import {
  appendRun,
  createActiveRun,
  exposeRunDebug,
  finishRun,
  loadActiveRun,
  loadRunHistory,
  recordRunStep,
  saveActiveRun,
  saveRunHistory,
  type ActiveRunRecord,
  type RunEndReason,
  type RunRecord,
} from './runLog'

export class GameSession {
  private config: GameConfig
  private sim: Simulation | null = null
  private stats: TickStats
  private dirty = true
  private seed = 42
  private activeRun: ActiveRunRecord | null = null
  private runHistory: RunRecord[]

  constructor(initialConfig: GameConfig = DEFAULT_GAME_CONFIG) {
    this.config = normalizeGameConfig(initialConfig)
    this.stats = createInitialStats(this.config)
    this.runHistory = loadRunHistory()

    const staleActiveRun = loadActiveRun()
    if (staleActiveRun && staleActiveRun.stepCount > 0) {
      const interrupted = finishRun(staleActiveRun, 'interrupted_reload')
      this.runHistory = appendRun(this.runHistory, interrupted)
      saveRunHistory(this.runHistory)
    }
    saveActiveRun(null)
    this.exposeRunDebug()
  }

  getConfig(): GameConfig {
    return this.config
  }

  getStats(): TickStats {
    return this.stats
  }

  getSimulation(): Simulation {
    if (!this.sim || this.dirty) {
      this.rebuildSimulation()
    }
    return this.sim as Simulation
  }

  getRunHistory(): readonly RunRecord[] {
    return this.runHistory
  }

  setConfig(update: Partial<GameConfig> | ((prev: GameConfig) => GameConfig)): GameConfig {
    const previous = this.config
    const next = typeof update === 'function'
      ? update(this.config)
      : { ...this.config, ...update }
    this.config = normalizeGameConfig(next)
    if (!sameConfig(previous, this.config)) {
      if (simConfigChanged(previous, this.config)) {
        this.finalizeActiveRun('config_changed')
        this.dirty = true
      }
    }
    return this.config
  }

  step(): TickStats {
    const simulation = this.getSimulation()
    this.stats = simulation.step()
    if (!this.activeRun) {
      this.activeRun = createActiveRun(this.config, this.seed, this.stats)
    }
    recordRunStep(this.activeRun, this.stats)
    saveActiveRun(this.activeRun)
    this.exposeRunDebug()
    return this.stats
  }

  reset(): TickStats {
    this.finalizeActiveRun('manual_reset')
    this.sim = null
    this.dirty = true
    this.stats = createInitialStats(this.config)
    return this.stats
  }

  dispose(reason: RunEndReason = 'session_dispose'): void {
    this.finalizeActiveRun(reason)
  }

  private rebuildSimulation(): void {
    this.finalizeActiveRun('session_rebuild')
    this.seed = Date.now() & 0xffffffff
    this.sim = new Simulation(toSimulationConfig(this.config), this.seed)
    this.stats = createInitialStats(this.config)
    this.activeRun = createActiveRun(this.config, this.seed, this.stats)
    saveActiveRun(this.activeRun)
    this.dirty = false
    this.exposeRunDebug()
  }

  private finalizeActiveRun(reason: RunEndReason): void {
    if (!this.activeRun) {
      return
    }
    if (this.activeRun.stepCount > 0) {
      this.runHistory = appendRun(this.runHistory, finishRun(this.activeRun, reason))
      saveRunHistory(this.runHistory)
    }
    this.activeRun = null
    saveActiveRun(null)
    this.exposeRunDebug()
  }

  private exposeRunDebug(): void {
    exposeRunDebug(this.runHistory, this.activeRun)
  }
}

function sameConfig(a: GameConfig, b: GameConfig): boolean {
  const keys = Object.keys(a) as Array<keyof GameConfig>
  return keys.every(key => a[key] === b[key])
}

/** Keys that only affect UI timing, not the simulation itself. */
const UI_ONLY_KEYS: ReadonlySet<keyof GameConfig> = new Set(['speed'])

function simConfigChanged(a: GameConfig, b: GameConfig): boolean {
  const keys = Object.keys(a) as Array<keyof GameConfig>
  return keys.some(key => !UI_ONLY_KEYS.has(key) && a[key] !== b[key])
}
